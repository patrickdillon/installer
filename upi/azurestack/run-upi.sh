#!/bin/bash

set -eux

# allows switching installer binary
openshift_install=openshift-install

rm -rf .openshift* auth ./*.ign
cp install-config.yaml.upi install-config.yaml

# shellcheck disable=SC1091
. envvar

# remove workers from the install config so the mco won't try to create them
# python3 -c '
# import yaml;
# path = "install-config.yaml";
# data = yaml.full_load(open(path));
# data["compute"][0]["replicas"] = 0;
# open(path, "w").write(yaml.dump(data, default_flow_style=False))'

${openshift_install} create manifests

# we don't want to create any machine* objects 
rm -f openshift/99_openshift-cluster-api_master-machines-*.yaml
rm -f openshift/99_openshift-cluster-api_worker-machineset-*.yaml


# typical upi instruction
python3 -c '
import yaml;
path = "manifests/cluster-scheduler-02-config.yml";
data = yaml.full_load(open(path));
data["spec"]["mastersSchedulable"] = False;
open(path, "w").write(yaml.dump(data, default_flow_style=False))'

# typical upi instruction
python3 -c '
import yaml;
path = "manifests/cluster-dns-02-config.yml";
data = yaml.full_load(open(path));
del data["spec"]["publicZone"];
open(path, "w").write(yaml.dump(data, default_flow_style=False))'

INFRA_ID=$(yq -r .status.infrastructureName manifests/cluster-infrastructure-02-config.yml)
RESOURCE_GROUP=$(yq -r .status.platformStatus.azure.resourceGroupName manifests/cluster-infrastructure-02-config.yml)

oc adm release extract "$OPENSHIFT_INSTALL_RELEASE_IMAGE_OVERRIDE" --credentials-requests --cloud=azure --to=credentials-request
files=$(ls credentials-request)
for f in $files
do
  SECRET_NAME=$(python3 -c 'import yaml;data = yaml.full_load(open("credentials-request/'${f}'"));print(data["spec"]["secretRef"]["name"])')
  SECRET_NAMESPACE=$(python3 -c 'import yaml;data = yaml.full_load(open("credentials-request/'${f}'"));print(data["spec"]["secretRef"]["namespace"])')
  filename=${f/request/secret}
  cat >> "manifests/$filename" << EOF
apiVersion: v1
kind: Secret
metadata:
    name: ${SECRET_NAME}
    namespace: ${SECRET_NAMESPACE}
stringData:
  azure_subscription_id: ${SUBSCRIPTION_ID}
  azure_client_id: ${AAD_CLIENT_ID}
  azure_client_secret: ${AAD_CLIENT_SECRET}
  azure_tenant_id: ${TENANT_ID}
  azure_resource_prefix: ${CLUSTER_NAME}
  azure_resourcegroup: ${RESOURCE_GROUP}
  azure_region: ${AZURE_REGION}
EOF
done

cat >> manifests/cco-configmap.yaml << EOF
apiVersion: v1
kind: ConfigMap
metadata:
  name: cloud-credential-operator-config
  namespace: openshift-cloud-credential-operator
  annotations:
    release.openshift.io/create-only: "true"
data:
  disabled: "true"
EOF

${openshift_install} create ignition-configs

az group create --name "$RESOURCE_GROUP" --location "$AZURE_REGION"

az storage account create -g "$RESOURCE_GROUP" --location "$AZURE_REGION" --name "${INFRA_ID}sa" --kind Storage --sku Standard_LRS
ACCOUNT_KEY=$(az storage account keys list -g "$RESOURCE_GROUP" --account-name "${INFRA_ID}sa" --query "[0].value" -o tsv)

az storage container create --name files --account-name "${INFRA_ID}sa" --public-access blob --account-key "$ACCOUNT_KEY"
az storage blob upload --account-name "${INFRA_ID}sa" --account-key "$ACCOUNT_KEY" -c "files" -f "bootstrap.ign" -n "bootstrap.ign"

az deployment group create -g "$RESOURCE_GROUP" \
  --template-file "01_vnet.json" \
  --parameters baseName="$INFRA_ID"

VHD_BLOB_URL="https://rhcossa.blob.ppe3.stackpoc.com/vhd/rhcos-49-84-202108221651.vhd"
az deployment group create -g "$RESOURCE_GROUP" \
  --template-file "02_storage.json" \
  --parameters vhdBlobURL="$VHD_BLOB_URL" \
  --parameters baseName="$INFRA_ID"

az deployment group create -g "$RESOURCE_GROUP" \
  --template-file "03_infra.json" \
  --parameters baseName="$INFRA_ID"

set +euE

az network dns zone create -g "$RESOURCE_GROUP" -n "${CLUSTER_NAME}.${BASE_DOMAIN}"
export PUBLIC_IP=$(az network public-ip list -g "$RESOURCE_GROUP" --query "[?name=='${INFRA_ID}-master-pip'] | [0].ipAddress" -o tsv)
az network dns record-set a add-record -g "$RESOURCE_GROUP" -z "${CLUSTER_NAME}.${BASE_DOMAIN}" -n api -a "$PUBLIC_IP" --ttl 60
az network dns record-set a add-record -g "$RESOURCE_GROUP" -z "${CLUSTER_NAME}.${BASE_DOMAIN}" -n api-int -a "$PUBLIC_IP" --ttl 60

export BOOTSTRAP_URL=$(az storage blob url --account-name "${INFRA_ID}sa" --account-key "$ACCOUNT_KEY" -c "files" -n "bootstrap.ign" -o tsv)
export BOOTSTRAP_IGNITION=$(jq -rcnM --arg v "3.2.0" --arg url "$BOOTSTRAP_URL" '{ignition:{version:$v,config:{replace:{source:$url}}}}' | base64 | tr -d '\n')

az deployment group create --verbose -g "$RESOURCE_GROUP" \
  --template-file "04_bootstrap.json" \
  --parameters bootstrapIgnition="$BOOTSTRAP_IGNITION" \
  --parameters sshKeyData="$SSH_KEY" \
  --parameters baseName="$INFRA_ID" \
  --parameters diagnosticsStorageAccountName="${INFRA_ID}sa"

set -euE

export MASTER_IGNITION=$(cat master.ign | base64 | tr -d '\n')
az deployment group create -g "$RESOURCE_GROUP" \
  --template-file "05_masters.json" \
  --parameters masterIgnition="$MASTER_IGNITION" \
  --parameters sshKeyData="$SSH_KEY" \
  --parameters baseName="$INFRA_ID" \
  --parameters diagnosticsStorageAccountName="${INFRA_ID}sa"

${openshift_install} wait-for bootstrap-complete --log-level debug

az network nsg rule delete -g "$RESOURCE_GROUP" --nsg-name "${INFRA_ID}"-nsg --name bootstrap_ssh_in
az vm stop -g "$RESOURCE_GROUP" --name "${INFRA_ID}"-bootstrap
az vm deallocate -g "$RESOURCE_GROUP" --name "${INFRA_ID}"-bootstrap
az vm delete -g "$RESOURCE_GROUP" --name "${INFRA_ID}"-bootstrap --yes
az disk delete -g "$RESOURCE_GROUP" --name "${INFRA_ID}"-bootstrap_OSDisk --no-wait --yes
az network nic delete -g "$RESOURCE_GROUP" --name "${INFRA_ID}"-bootstrap-nic --no-wait
az storage blob delete --account-key "$ACCOUNT_KEY" --account-name "${INFRA_ID}sa" --container-name files --name bootstrap.ign
az network public-ip delete -g "$RESOURCE_GROUP" --name "${INFRA_ID}"-bootstrap-ssh-pip

export WORKER_IGNITION=$(cat worker.ign | base64 | tr -d '\n')
az deployment group create -g "$RESOURCE_GROUP" \
  --template-file "06_workers.json" \
  --parameters workerIgnition="$WORKER_IGNITION" \
  --parameters sshKeyData="$SSH_KEY" \
  --parameters baseName="$INFRA_ID" \
  --parameters diagnosticsStorageAccountName="${INFRA_ID}sa"

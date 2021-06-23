#!/bin/bash

set -eux

# allows switching installer binary
openshift_install=openshift-install

rm -rf .openshift* auth ./*.ign
cp install-config.yaml.upi install-config.yaml

# shellcheck disable=SC1091
. envvar

# remove workers from the install config so the mco won't try to create them
python3 -c '
import yaml;
path = "install-config.yaml";
data = yaml.full_load(open(path));
data["compute"][0]["replicas"] = 0;
open(path, "w").write(yaml.dump(data, default_flow_style=False))'

azurestackjson=$(cat <<EOF
{
  "name": "AzureStackCloud",
  "resourceManagerEndpoint": "https://management.ppe3.stackpoc.com",
  "activeDirectoryEndpoint": "https://login.microsoftonline.com/",
  "galleryEndpoint": "https://providers.ppe3.local:30016/",
  "storageEndpointSuffix": "https://providers.ppe3.local:30016/",
  "serviceManagementEndpoint": "https://management.stackpoc.com/81c9b804-ec9e-4b5a-8845-1d197268b1f5",
  "graphEndpoint":                "https://graph.windows.net/",
  "resourceIdentifiers": {
    "graph": "https://graph.windows.net/"
  }
}
EOF
)

${openshift_install} create manifests

# we don't want to create any machine* objects 
rm -f openshift/99_openshift-cluster-api_master-machines-*.yaml
rm -f openshift/99_openshift-cluster-api_worker-machineset-*.yaml

python3 -c "
import sys, json, yaml, os;
path = 'manifests/cloud-provider-config.yaml';
data = yaml.full_load(open(path));
config = json.loads(data['data']['config']);
config['cloud'] = 'AzureStackCloud';
config['tenantId'] = os.environ['TENANT_ID'];
config['subscriptionId'] = os.environ['SUBSCRIPTION_ID'];
config['location'] = os.environ['AZURE_REGION'];
config['aadClientId'] = os.environ['AAD_CLIENT_ID'];
config['aadClientSecret'] = os.environ['AAD_CLIENT_SECRET'];
config['useManagedIdentityExtension'] = False;
config['useInstanceMetadata'] = False;
config['loadBalancerSku'] = 'basic';
data['data']['config'] = json.dumps(config);
data['data']['endpoints'] = json.dumps($azurestackjson);
open(path, 'w').write(yaml.dump(data, default_flow_style=False))"

python3 -c '
import yaml,os;
path = "manifests/cluster-config.yaml";
data = yaml.full_load(open(path));
install_config = yaml.full_load(data["data"]["install-config"]);
platform = install_config["platform"];
platform["azure"]["cloudName"] = "AzureStackCloud";
platform["azure"]["region"] = os.environ["AZURE_REGION"];
install_config["platform"] = platform;
data["data"]["install-config"] = yaml.dump(install_config);
open(path, "w").write(yaml.dump(data, default_flow_style=False))'

python3 -c '
import base64, yaml,os;
path = "openshift/99_cloud-creds-secret.yaml";
data = yaml.full_load(open(path));
data["data"]["azure_subscription_id"] = base64.b64encode(os.environ["SUBSCRIPTION_ID"].encode("ascii")).decode("ascii");
data["data"]["azure_client_id"] = base64.b64encode(os.environ["AAD_CLIENT_ID"].encode("ascii")).decode("ascii");
data["data"]["azure_client_secret"] = base64.b64encode(os.environ["AAD_CLIENT_SECRET"].encode("ascii")).decode("ascii");
data["data"]["azure_tenant_id"] = base64.b64encode(os.environ["TENANT_ID"].encode("ascii")).decode("ascii");
data["data"]["azure_region"] = base64.b64encode(os.environ["AZURE_REGION"].encode("ascii")).decode("ascii");
open(path, "w").write(yaml.dump(data, default_flow_style=False))'

python3 -c '
import yaml;
path = "manifests/cluster-infrastructure-02-config.yml";
data = yaml.full_load(open(path));
data["status"]["platformStatus"]["azure"]["cloudName"] = "AzureStackCloud";
open(path, "w").write(yaml.dump(data, default_flow_style=False))'

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
del data["spec"]["privateZone"];
open(path, "w").write(yaml.dump(data, default_flow_style=False))'

cat << EOF > openshift/99_openshift-machineconfig_99-master-azurestackcloud.yaml
apiVersion: machineconfiguration.openshift.io/v1
kind: MachineConfig
metadata:
  creationTimestamp: null
  labels:
    machineconfiguration.openshift.io/role: master
  name: 99-master-azurestack
spec:
  config:
    ignition:
      config: {}
      security:
        tls: {}
      timeouts: {}
      version: 3.2.0
    networkd: {}
    passwd: {}
    storage:
      files:
        - path: /etc/kubernetes/azurestackcloud.json
          contents:
            source: data:text/plain;charset=utf-8;base64,$(echo $azurestackjson | base64 | tr -d '\n')
          mode: 420
          user:
            name: root
    systemd:
      units:
        - name: kubelet.service
          dropins:
            - name: 10-azurestack.conf
              contents: |
                [Service]
                Environment="AZURE_ENVIRONMENT_FILEPATH=/etc/kubernetes/azurestackcloud.json"
  fips: false
  kernelArguments: null
  kernelType: ""
  osImageURL: ""
EOF

cat << EOF > openshift/99_openshift-machineconfig_99-worker-azurestackcloud.yaml
apiVersion: machineconfiguration.openshift.io/v1
kind: MachineConfig
metadata:
  creationTimestamp: null
  labels:
    machineconfiguration.openshift.io/role: worker
  name: 99-worker-azurestack
spec:
  config:
    ignition:
      config: {}
      security:
        tls: {}
      timeouts: {}
      version: 3.2.0
    networkd: {}
    passwd: {}
    storage:
      files:
        - path: /etc/kubernetes/azurestackcloud.json
          contents:
            source: data:text/plain;charset=utf-8;base64,$(echo $azurestackjson | base64 | tr -d '\n')
          mode: 420
          user:
            name: root
    systemd:
      units:
        - name: kubelet.service
          dropins:
            - name: 10-azurestack.conf
              contents: |
                [Service]
                Environment="AZURE_ENVIRONMENT_FILEPATH=/etc/kubernetes/azurestackcloud.json"
  fips: false
  kernelArguments: null
  kernelType: ""
  osImageURL: ""
EOF

INFRA_ID=$(yq r manifests/cluster-infrastructure-02-config.yml 'status.infrastructureName')
RESOURCE_GROUP=$(yq r manifests/cluster-infrastructure-02-config.yml 'status.platformStatus.azure.resourceGroupName')

${openshift_install} create ignition-configs

az group create --name "$RESOURCE_GROUP" --location "$AZURE_REGION"

az storage account create -g "$RESOURCE_GROUP" --location "$AZURE_REGION" --name "${INFRA_ID}sa" --kind Storage --sku Standard_LRS
ACCOUNT_KEY=$(az storage account keys list -g "$RESOURCE_GROUP" --account-name "${INFRA_ID}sa" --query "[0].value" -o tsv)

az storage container create --name files --account-name "${INFRA_ID}sa" --public-access blob --account-key "$ACCOUNT_KEY"
az storage blob upload --account-name "${INFRA_ID}sa" --account-key "$ACCOUNT_KEY" -c "files" -f "bootstrap.ign" -n "bootstrap.ign"


az deployment group create -g "$RESOURCE_GROUP" \
  --template-file "01_vnet.json" \
  --parameters baseName="$INFRA_ID"


VHD_BLOB_URL="https://rhcossa.blob.ppe3.stackpoc.com/vhd/rhcos4.vhd"
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
export BOOTSTRAP_IGNITION=$(jq -rcnM --arg v "3.1.0" --arg url "$BOOTSTRAP_URL" '{ignition:{version:$v,config:{replace:{source:$url}}}}' | base64 | tr -d '\n')

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
  --parameters masterVMSize="Standard_D4_v2" \
  --parameters diskSizeGB="1023" \
  --parameters diagnosticsStorageAccountName="${INFRA_ID}sa"

export WORKER_IGNITION=$(cat worker.ign | base64 | tr -d '\n')
az deployment group create -g "$RESOURCE_GROUP" \
  --template-file "06_workers.json" \
  --parameters workerIgnition="$WORKER_IGNITION" \
  --parameters sshKeyData="$SSH_KEY" \
  --parameters baseName="$INFRA_ID" \
  --parameters diagnosticsStorageAccountName="${INFRA_ID}sa"

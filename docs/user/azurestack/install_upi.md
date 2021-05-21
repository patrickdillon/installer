# Azure Stack Hub UPI Installation in the PPE Environment

This is a guide on how to install OpenShift on Azure Stack Hub in the PPE environment. This document is intended to be used for development
purposes and does not represent an officially supported installation method. Because of the intended audience & purpose of this document, 
it will be briefer, less contextual, and more informal than a standard UPI guide. The standard Azure UPI guide may be helpful to provide more
context.

## Quick Start (using shell script)

* [Prereqs](#Prerequisites)
* [Set Environment Variables](#Set-environment-variables)
* [Run Shell Script](#Script)

## Prerequisites

### yq  

There seem to be a few versions of `yq` floating around with different syntaxes. This guide uses https://github.com/mikefarah/yq so either use that one or adjust the commands to fit your preferred `yq`.

### Connecting to the PPE Environment with the Azure CLI

These commands will register the PPE cloud environment and log you in.

```console
az cloud register \
    -n PPE \
    --endpoint-resource-manager "https://management.ppe3.stackpoc.com" \
    --suffix-storage-endpoint "ppe3.stackpoc.com" 
az cloud set -n PPE
az cloud update --profile 2019-03-01-hybrid
az login --tenant 36ad9d54-ca17-4ea7-89b3-0e1638cf878e
```

You can switch between clouds: e.g. `az cloud set -n AzureCloud` and `az cloud set -n PPE`.

`az cloud show` will indicate the current cloud. Obviously you need to be on PPE for these instructions.

### Create a service principal

You only need to generate your credentials once:

```console
az ad sp create-for-rbac --role Owner --name <username>-upi > sp.json
```

### Create an install config

Create an install config. The Installer has not merged most azurestack support yet, so you should use credentials for public Azure and also create   install config for public Azure. The commands in following steps will substitute the appropriate values for azure stack into the manifests.

*NOTE* You can use base domain `ppe.ash.devcluster.openshift.com`. There is an NS record setup in route53
to delegate this zone to PPE nameservers.

```console
$ openshift-install create install-config
? SSH Public Key /home/user_id/.ssh/id_rsa.pub
? Platform azure
? azure subscription id xxxxxxxx-xxxx-xxxx-xxxx-xxxxxxxxxxxx
? azure tenant id xxxxxxxx-xxxx-xxxx-xxxx-xxxxxxxxxxxx
? azure service principal client id xxxxxxxx-xxxx-xxxx-xxxx-xxxxxxxxxxxx
? azure service principal client secret xxxxxxxx-xxxx-xxxx-xxxx-xxxxxxxxxxxx
INFO Saving user credentials to "/home/user_id/.azure/osServicePrincipal.json"
? Region centralus
? Base Domain ppe.ash.devcluster.openshift.com
? Cluster Name test
? Pull Secret [? for help]
```

## Set Environment Variables

Some data from the install configuration file will be used on later steps. Export them as environment variables with:

```sh
export CLUSTER_NAME=$(yq r install-config.yaml metadata.name)
export AZURE_REGION=ppe3
export SSH_KEY=$(yq r install-config.yaml sshKey | xargs)
export BASE_DOMAIN=$(yq r install-config.yaml baseDomain)
export TENANT_ID=$(az account show | jq -r .tenantId)
export SUBSCRIPTION_ID=$(az account show | jq -r .id)
export AAD_CLIENT_ID=$(jq -r .appId sp.json)
export AAD_CLIENT_SECRET=$(jq -r .password sp.json)

# currently we need to use images built with openshift/cluster-config-operator#186
# as well as a kubelet patched with https://github.com/openshift/kubernetes/pull/643
# this pr has been built into a machine-os-content container and a patched release image is available at:
export OPENSHIFT_INSTALL_RELEASE_IMAGE_OVERRIDE=http://quay.io/padillon/ash-release-mirror:latest
```

If you are using the script, you can set these variables in the [envvar](../../../upi/azurestack/envvar) file.

## Script

All of the commands below are included in [a shell script](../../../upi/azurestack/run-upi.sh).

This script expects:

* install config saved as `install-config.yaml.upi` as a template for multiple invocations.
* an `envvar` file with [the variables above](#set-environment-variables).
* public azure credentials in ~/.azure/osServicePrincipal.json

```console
$ ls envvar install-config.yaml.upi 
envvar  install-config.yaml.upi
$ ./run-upi.sh
+ openshift_install=openshift-install
+ rm -rf '.openshift*' auth './*.ign'
+ cp install-config.yaml.upi install-config.yaml
+ . envvar
[...]
```

After the scripts have completed worker machines should be created but not yet registered as nodes.

Approve the pending CSRs according to [the normal UPI instructions](../azure/install_upi.md#approve-the-worker-csrs): `oc get csr -A` and then `oc adm certificate approve <worker-0-csr> <worker-1-csr> <worker-2-csr>`.

Running the script should handle all of the installation (except approving worker CSRs as noted above). So you can stop here, or read on for more context of what the script is doing.


## Empty the compute pool

We'll be providing the compute machines ourselves, so edit the resulting `install-config.yaml` to set `replicas` to 0 for the `compute` pool:

```sh
python3 -c '
import yaml;
path = "install-config.yaml";
data = yaml.full_load(open(path));
data["compute"][0]["replicas"] = 0;
open(path, "w").write(yaml.dump(data, default_flow_style=False))'
```

## Create manifests

Create manifests to enable customizations that are not exposed via the install configuration.

```console
$ openshift-install create manifests
INFO Credentials loaded from file "/home/user_id/.azure/osServicePrincipal.json"
INFO Consuming "Install Config" from target directory
WARNING Making control-plane schedulable by setting MastersSchedulable to true for Scheduler cluster settings
```

### Remove control plane machines and machinesets

Remove the control plane machines and compute machinesets from the manifests.
We'll be providing those ourselves and don't want to involve the [machine-API operator][machine-api-operator].

```sh
rm -f openshift/99_openshift-cluster-api_master-machines-*.yaml
rm -f openshift/99_openshift-cluster-api_worker-machineset-*.yaml
```

### Edit Manifests for Azure Stack Hub specific content

The Installer does not yet support generating manifests specifically for Azure Stack Hub so we need to manually edit the contents to
substitute in our environment variabes.

First we edit the cloud provider config. There is active work in progress so that ultimately the secret will not be stored here.

```sh
python3 -c '
import sys, json, yaml, os;
path = "manifests/cloud-provider-config.yaml";
data = yaml.full_load(open(path));
config = json.loads(data["data"]["config"]);
config["cloud"] = "AzureStackCloud";
config["tenantId"] = os.environ["TENANT_ID"];
config["subscriptionId"] = os.environ["SUBSCRIPTION_ID"];
config["location"] = os.environ["AZURE_REGION"];
config["aadClientId"] = os.environ["AAD_CLIENT_ID"];
config["aadClientSecret"] = os.environ["AAD_CLIENT_SECRET"];
config["useManagedIdentityExtension"] = False;
config["useInstanceMetadata"] = False;
config["loadBalancerSku"] = "basic";
data["data"]["config"] = json.dumps(config);
open(path, "w").write(yaml.dump(data, default_flow_style=False))'
```

Then set the region and cloud for the cluster config operator.

```sh
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
```

You may be using Azure credentials, so overwrite those with Azure Stack credentials.

```sh
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
```

Set the cloud in the infrastructure manifest.

```sh
python3 -c '
import yaml;
path = "manifests/cluster-infrastructure-02-config.yml";
data = yaml.full_load(open(path));
data["status"]["platformStatus"]["azure"]["cloudName"] = "AzureStackCloud";
open(path, "w").write(yaml.dump(data, default_flow_style=False))'
```

### Make control-plane nodes unschedulable

Currently [emptying the compute pools](#empty-the-compute-pool) makes control-plane nodes schedulable.
But due to a [Kubernetes limitation][kubernetes-service-load-balancers-exclude-masters], router pods running on control-plane nodes will not be reachable by the ingress load balancer.
Update the scheduler configuration to keep router pods and other workloads off the control-plane nodes:

```sh
python3 -c '
import yaml;
path = "manifests/cluster-scheduler-02-config.yml";
data = yaml.full_load(open(path));
data["spec"]["mastersSchedulable"] = False;
open(path, "w").write(yaml.dump(data, default_flow_style=False))'
```

### Remove DNS Zones

We don't want [the ingress operator][ingress-operator] to create DNS records (we're going to do it manually) so we need to remove
the `privateZone` and `publicZone` sections from the DNS configuration in manifests.

```sh
python3 -c '
import yaml;
path = "manifests/cluster-dns-02-config.yml";
data = yaml.full_load(open(path));
del data["spec"]["publicZone"];
del data["spec"]["privateZone"];
open(path, "w").write(yaml.dump(data, default_flow_style=False))'
```

### Create Machine Cofigs
We will create an azurestackcloud.json file to hold the ASH specific endpoints. We will set an environment variable so the kubelet knows where to find the file:

```sh
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

```


### Resource Group Name and Infra ID

The OpenShift cluster has been assigned an identifier in the form of `<cluster_name>-<random_string>`. This identifier, called "Infra ID", will be used as
the base name of most resources that will be created in this example. Export the Infra ID as an environment variable that will be used later in this example:

```sh
export INFRA_ID=$(yq r manifests/cluster-infrastructure-02-config.yml 'status.infrastructureName')
```

Also, all resources created in this Azure deployment will exist as part of a [resource group][azure-resource-group]. The resource group name is also
based on the Infra ID, in the form of `<cluster_name>-<random_string>-rg`. Export the resource group name to an environment variable that will be user later:

```sh
export RESOURCE_GROUP=$(yq r manifests/cluster-infrastructure-02-config.yml 'status.platformStatus.azure.resourceGroupName')
```

## Create ignition configs

Now we can create the bootstrap ignition configs:

```console
$ openshift-install create ignition-configs
INFO Consuming Openshift Manifests from target directory
INFO Consuming Worker Machines from target directory
INFO Consuming Common Manifests from target directory
INFO Consuming Master Machines from target directory
```

After running the command, several files will be available in the directory.

```console
$ tree
.
├── auth
│   └── kubeconfig
├── bootstrap.ign
├── master.ign
├── metadata.json
└── worker.ign
```

## Create The Resource Group and identity

Use the command below to create the resource group in the selected Azure region:

```sh
az group create --name "$RESOURCE_GROUP" --location "$AZURE_REGION"
```

## Upload the files to a Storage Account

The deployment steps will read the Red Hat Enterprise Linux CoreOS virtual hard disk (VHD) image and the bootstrap ignition config file
from a blob. Create a storage account that will be used to store them and export its key as an environment variable.

```sh
az storage account create -g "$RESOURCE_GROUP" --location "$AZURE_REGION" --name "${INFRA_ID}sa" --kind Storage --sku Standard_LRS
export ACCOUNT_KEY=$(az storage account keys list -g "$RESOURCE_GROUP" --account-name "${INFRA_ID}sa" --query "[0].value" -o tsv)
```

### Upload the bootstrap ignition

Create a blob storage container and upload the generated `bootstrap.ign` file:

```sh
az storage container create --name files --account-name "${INFRA_ID}sa" --public-access blob --account-key "$ACCOUNT_KEY"
az storage blob upload --account-name "${INFRA_ID}sa" --account-key "$ACCOUNT_KEY" -c "files" -f "bootstrap.ign" -n "bootstrap.ign"
```

### Copy the cluster image

Because there are no publicly available ASH RHCOS images yet, we will use an existing image in the environment:

```sh
export VHD_BLOB_URL="https://rhcossa.blob.ppe3.stackpoc.com/vhd/rhcos4.vhd"
```

## Create the DNS zones

A few DNS records are required for clusters that use user-provisioned infrastructure. Feel free to choose the DNS strategy that fits you best.

In this example we're going to use [Azure's own DNS solution][azure-dns], so we're going to create a new public DNS zone for external (internet) visibility.
Note that the public zone don't necessarily need to exist in the same resource group of the
cluster deployment itself and may even already exist in your organization for the desired base domain. If that's the case, you can skip the public DNS
zone creation step, but make sure the install config generated earlier [reflects that scenario](customization.md#cluster-scoped-properties).

Create the new *public* DNS zone in the resource group exported in the `BASE_DOMAIN_RESOURCE_GROUP` environment variable, or just skip this step if you're going
to use one that already exists in your organization:

```sh
az network dns zone create -g "$RESOURCE_GROUP" -n "${CLUSTER_NAME}.${BASE_DOMAIN}"
```

## Deployment

The key part of this UPI deployment are the [Azure Resource Manager][azuretemplates] templates, which are responsible
for deploying most resources. They're provided as a few json files named following the "NN_name.json" pattern. In the
next steps we're going to deploy each one of them in order, using [az (Azure CLI)][azurecli] and providing the expected parameters.

## Deploy the Virtual Network

In this example we're going to create a Virtual Network and subnets specifically for the OpenShift cluster. You can skip this step
if the cluster is going to live in a VNet already existing in your organization, or you can edit the `01_vnet.json` file to your
own needs (e.g. change the subnets address prefixes in CIDR format).

Copy the [`01_vnet.json`](../../../upi/azurestack/01_vnet.json) ARM template locally.

Create the deployment using the `az` client:

```sh
az deployment group create -g $RESOURCE_GROUP \
  --template-file "01_vnet.json" \
  --parameters baseName="$INFRA_ID"
```

## Deploy the image

Copy the [`02_storage.json`](../../../upi/azurestack/02_storage.json) ARM template locally.

Create the deployment using the `az` client:

```sh
az deployment group create -g $RESOURCE_GROUP \
  --template-file "02_storage.json" \
  --parameters vhdBlobURL="$VHD_BLOB_URL" \
  --parameters baseName="$INFRA_ID"
```

## Deploy the load balancers

Copy the [`03_infra.json`](../../../upi/azurestack/03_infra.json) ARM template locally.

Deploy the load balancers and public IP addresses using the `az` client:

```sh
az deployment group create -g $RESOURCE_GROUP \
  --template-file "03_infra.json" \
  --parameters privateDNSZoneName="${CLUSTER_NAME}.${BASE_DOMAIN}" \
  --parameters baseName="$INFRA_ID"
```

Create an `api` DNS record in the *public* zone for the API public load balancer. Note that the `BASE_DOMAIN_RESOURCE_GROUP` must point to the resource group where the public DNS zone exists.

```sh
export PUBLIC_IP=$(az network public-ip list -g "$RESOURCE_GROUP" --query "[?name=='${INFRA_ID}-master-pip'] | [0].ipAddress" -o tsv)
az network dns record-set a add-record -g "$RESOURCE_GROUP" -z "${CLUSTER_NAME}.${BASE_DOMAIN}" -n api -a "$PUBLIC_IP" --ttl 60
az network dns record-set a add-record -g "$RESOURCE_GROUP" -z "${CLUSTER_NAME}.${BASE_DOMAIN}" -n api-int -a "$PUBLIC_IP" --ttl 60
```

## Launch the temporary cluster bootstrap

Copy the [`04_bootstrap.json`](../../../upi/azurestack/04_bootstrap.json) ARM template locally.

Create the deployment using the `az` client:

```sh
export BOOTSTRAP_URL=$(az storage blob url --account-name "${INFRA_ID}sa" --account-key "$ACCOUNT_KEY" -c "files" -n "bootstrap.ign" -o tsv)
export BOOTSTRAP_IGNITION=$(jq -rcnM --arg v "3.1.0" --arg url "$BOOTSTRAP_URL" '{ignition:{version:$v,config:{replace:{source:$url}}}}' | base64 | tr -d '\n')

az deployment group create --verbose -g "$RESOURCE_GROUP" \
  --template-file "04_bootstrap.json" \
  --parameters bootstrapIgnition="$BOOTSTRAP_IGNITION" \
  --parameters sshKeyData="$SSH_KEY" \
  --parameters baseName="$INFRA_ID" \
  --parameters diagnosticsStorageAccountName="${INFRA_ID}sa"
```

## Launch the permanent control plane

Copy the [`05_masters.json`](../../../upi/azurestack/05_masters.json) ARM template locally.

Create the deployment using the `az` client:

```sh
export MASTER_IGNITION=$(cat master.ign | base64 | tr -d '\n')

az deployment group create -g "$RESOURCE_GROUP" \
  --template-file "05_masters.json" \
  --parameters masterIgnition="$MASTER_IGNITION" \
  --parameters sshKeyData="$SSH_KEY" \
  --parameters baseName="$INFRA_ID" \
  --parameters masterVMSize="Standard_D4_v2" \
  --parameters diskSizeGB="1023" \
  --parameters diagnosticsStorageAccountName="${INFRA_ID}sa"
```

[azuretemplates]: https://docs.microsoft.com/en-us/azure/azure-resource-manager/template-deployment-overview
[openshiftinstall]: https://github.com/openshift/installer
[azurecli]: https://docs.microsoft.com/en-us/cli/azure/
[jqjson]: https://stedolan.github.io/jq/
[yqyaml]: https://yq.readthedocs.io/en/latest/
[ingress-operator]: https://github.com/openshift/cluster-ingress-operator
[machine-api-operator]: https://github.com/openshift/machine-api-operator
[azure-identity]: https://docs.microsoft.com/en-us/azure/architecture/framework/security/identity
[azure-resource-group]: https://docs.microsoft.com/en-us/azure/azure-resource-manager/management/overview#resource-groups
[azure-dns]: https://docs.microsoft.com/en-us/azure/dns/dns-overview
[kubernetes-service-load-balancers-exclude-masters]: https://github.com/kubernetes/kubernetes/issues/65618

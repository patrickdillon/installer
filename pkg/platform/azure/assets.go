package azure

import (
	"bytes"
	"encoding/json"
	"github.com/pkg/errors"
)

//CloudProviderConfig is the azure cloud provider config
type CloudProviderConfig struct {
	TenantID       string
	SubscriptionID string
	GroupLocation  string
	ResourcePrefix string
}

// CloudProviderConfig provides the cloud provider config file for Microsoft Azure.
func (a *Azure) CloudProviderConfig(infraID, clusterName string) (string, error) {
	session, err := GetSession()
	if err != nil {
		return "", errors.Wrap(err, "could not get azure session")
	}
	azureConfig, err := CloudProviderConfig{
		GroupLocation:  a.Region,
		ResourcePrefix: infraID,
		SubscriptionID: session.Credentials.SubscriptionID,
		TenantID:       session.Credentials.TenantID,
	}.JSON()
	if err != nil {
		return "", errors.Wrap(err, "could not create cloud provider config")
	}
	return azureConfig, nil
}

// JSON generates the cloud provider json config for the azure platform.
// managed resource names are matching the convention defined by capz
func (params CloudProviderConfig) JSON() (string, error) {
	resourceGroupName := params.ResourcePrefix + "-rg"
	config := config{
		authConfig: authConfig{
			Cloud:                       "AzurePublicCloud",
			TenantID:                    params.TenantID,
			SubscriptionID:              params.SubscriptionID,
			UseManagedIdentityExtension: true,
			// The cloud provider needs the clientID which is only known after terraform has run.
			// When left empty, the existing managed identity on the VM will be used.
			// By leaving it empty, we don't have to create the identity before running the installer.
			// We only need to know that there will be one assigned to the VM, and we control this.
			// ref: https://github.com/kubernetes/kubernetes/blob/4b7c607ba47928a7be77fadef1550d6498397a4c/staging/src/k8s.io/legacy-cloud-providers/azure/auth/azure_auth.go#L69
			UserAssignedIdentityID: "",
		},
		ResourceGroup:          resourceGroupName,
		Location:               params.GroupLocation,
		SubnetName:             params.ResourcePrefix + "-node-subnet",
		SecurityGroupName:      params.ResourcePrefix + "-node-nsg",
		VnetName:               params.ResourcePrefix + "-vnet",
		VnetResourceGroup:      resourceGroupName,
		RouteTableName:         params.ResourcePrefix + "-node-routetable",
		CloudProviderBackoff:   true,
		CloudProviderRateLimit: true,

		// The default rate limits for Azure cloud provider are https://github.com/kubernetes/kubernetes/blob/f8d2b6b982bb06fc64979ac53ae668284d9c003c/staging/src/k8s.io/legacy-cloud-providers/azure/azure.go#L51-L56
		// While the AKS recommends following rate limits for large clusters https://github.com/Azure/aks-engine/blob/0f6aa91fa1870d5be657c62374d11f7d6009121d/examples/largeclusters/kubernetes.json#L9-L15
		// 									default		AKS (large)	Change
		// cloudProviderBackoffRetries		6			6					NO
		// cloudProviderBackoffJitter		1.0			1					NO
		// cloudProviderBackoffExponent		1.5			1.5					NO
		// cloudProviderBackoffDuration		5			6					YES to 6
		// cloudProviderRateLimitQPS		3			3					YES to 6
		// cloudProviderRateLimitBucket		5			10					YES to 10
		CloudProviderBackoffDuration:      6,
		CloudProviderRateLimitQPS:         6,
		CloudProviderRateLimitQPSWrite:    6,
		CloudProviderRateLimitBucket:      10,
		CloudProviderRateLimitBucketWrite: 10,

		UseInstanceMetadata: true,
		//default to standard load balancer, supports tcp resets on idle
		//https://docs.microsoft.com/en-us/azure/load-balancer/load-balancer-tcp-reset
		LoadBalancerSku: "standard",
	}
	buff := &bytes.Buffer{}
	encoder := json.NewEncoder(buff)
	encoder.SetIndent("", "\t")
	if err := encoder.Encode(config); err != nil {
		return "", err
	}
	return buff.String(), nil
}

//authConfig is part of the CloudProviderConfig as defined in https://github.com/kubernetes/kubernetes/blob/v1.13.5/pkg/cloudprovider/providers/azure/auth/azure_auth.go#L32
type authConfig struct {
	// The cloud environment identifier. Takes values from https://github.com/Azure/go-autorest/blob/ec5f4903f77ed9927ac95b19ab8e44ada64c1356/autorest/azure/environments.go#L13
	Cloud string `json:"cloud" yaml:"cloud"`
	// The AAD Tenant ID for the Subscription that the cluster is deployed in
	TenantID string `json:"tenantId" yaml:"tenantId"`
	// The ClientID for an AAD application with RBAC access to talk to Azure RM APIs
	AADClientID string `json:"aadClientId" yaml:"aadClientId"`
	// The ClientSecret for an AAD application with RBAC access to talk to Azure RM APIs
	AADClientSecret string `json:"aadClientSecret" yaml:"aadClientSecret"`
	// The path of a client certificate for an AAD application with RBAC access to talk to Azure RM APIs
	AADClientCertPath string `json:"aadClientCertPath" yaml:"aadClientCertPath"`
	// The password of the client certificate for an AAD application with RBAC access to talk to Azure RM APIs
	AADClientCertPassword string `json:"aadClientCertPassword" yaml:"aadClientCertPassword"`
	// Use managed service identity for the virtual machine to access Azure ARM APIs
	UseManagedIdentityExtension bool `json:"useManagedIdentityExtension" yaml:"useManagedIdentityExtension"`
	// UserAssignedIdentityID contains the Client ID of the user assigned MSI which is assigned to the underlying VMs. If empty the user assigned identity is not used.
	// More details of the user assigned identity can be found at: https://docs.microsoft.com/en-us/azure/active-directory/managed-service-identity/overview
	// For the user assigned identity specified here to be used, the UseManagedIdentityExtension has to be set to true.
	UserAssignedIdentityID string `json:"userAssignedIdentityID" yaml:"userAssignedIdentityID"`
	// The ID of the Azure Subscription that the cluster is deployed in
	SubscriptionID string `json:"subscriptionId" yaml:"subscriptionId"`
}

//config is the cloud provider config as defined in https://github.com/kubernetes/kubernetes/blob/v1.13.5/pkg/cloudprovider/providers/azure/azure.go#L81
type config struct {
	authConfig

	// The name of the resource group that the cluster is deployed in
	ResourceGroup string `json:"resourceGroup" yaml:"resourceGroup"`
	// The location of the resource group that the cluster is deployed in
	Location string `json:"location" yaml:"location"`
	// The name of the VNet that the cluster is deployed in
	VnetName string `json:"vnetName" yaml:"vnetName"`
	// The name of the resource group that the Vnet is deployed in
	VnetResourceGroup string `json:"vnetResourceGroup" yaml:"vnetResourceGroup"`
	// The name of the subnet that the cluster is deployed in
	SubnetName string `json:"subnetName" yaml:"subnetName"`
	// The name of the security group attached to the cluster's subnet
	SecurityGroupName string `json:"securityGroupName" yaml:"securityGroupName"`
	// (Optional in 1.6) The name of the route table attached to the subnet that the cluster is deployed in
	RouteTableName string `json:"routeTableName" yaml:"routeTableName"`
	// (Optional) The name of the availability set that should be used as the load balancer backend
	// If this is set, the Azure cloudprovider will only add nodes from that availability set to the load
	// balancer backend pool. If this is not set, and multiple agent pools (availability sets) are used, then
	// the cloudprovider will try to add all nodes to a single backend pool which is forbidden.
	// In other words, if you use multiple agent pools (availability sets), you MUST set this field.
	PrimaryAvailabilitySetName string `json:"primaryAvailabilitySetName" yaml:"primaryAvailabilitySetName"`
	// The type of azure nodes. Candidate values are: vmss and standard.
	// If not set, it will be default to standard.
	VMType string `json:"vmType" yaml:"vmType"`
	// The name of the scale set that should be used as the load balancer backend.
	// If this is set, the Azure cloudprovider will only add nodes from that scale set to the load
	// balancer backend pool. If this is not set, and multiple agent pools (scale sets) are used, then
	// the cloudprovider will try to add all nodes to a single backend pool which is forbidden.
	// In other words, if you use multiple agent pools (scale sets), you MUST set this field.
	PrimaryScaleSetName string `json:"primaryScaleSetName" yaml:"primaryScaleSetName"`
	// Enable exponential backoff to manage resource request retries
	CloudProviderBackoff bool `json:"cloudProviderBackoff" yaml:"cloudProviderBackoff"`
	// Backoff retry limit
	CloudProviderBackoffRetries int `json:"cloudProviderBackoffRetries" yaml:"cloudProviderBackoffRetries"`
	// Backoff exponent
	CloudProviderBackoffExponent float64 `json:"cloudProviderBackoffExponent" yaml:"cloudProviderBackoffExponent"`
	// Backoff duration
	CloudProviderBackoffDuration int `json:"cloudProviderBackoffDuration" yaml:"cloudProviderBackoffDuration"`
	// Backoff jitter
	CloudProviderBackoffJitter float64 `json:"cloudProviderBackoffJitter" yaml:"cloudProviderBackoffJitter"`
	// Enable rate limiting
	CloudProviderRateLimit bool `json:"cloudProviderRateLimit" yaml:"cloudProviderRateLimit"`
	// Rate limit QPS (Read)
	CloudProviderRateLimitQPS float32 `json:"cloudProviderRateLimitQPS" yaml:"cloudProviderRateLimitQPS"`
	// Rate limit Bucket Size
	CloudProviderRateLimitBucket int `json:"cloudProviderRateLimitBucket" yaml:"cloudProviderRateLimitBucket"`
	// Rate limit QPS (Write)
	CloudProviderRateLimitQPSWrite float32 `json:"cloudProviderRateLimitQPSWrite" yaml:"cloudProviderRateLimitQPSWrite"`
	// Rate limit Bucket Size
	CloudProviderRateLimitBucketWrite int `json:"cloudProviderRateLimitBucketWrite" yaml:"cloudProviderRateLimitBucketWrite"`

	// Use instance metadata service where possible
	UseInstanceMetadata bool `json:"useInstanceMetadata" yaml:"useInstanceMetadata"`

	// Sku of Load Balancer and Public IP. Candidate values are: basic and standard.
	// If not set, it will be default to basic.
	LoadBalancerSku string `json:"loadBalancerSku" yaml:"loadBalancerSku"`
	// ExcludeMasterFromStandardLB excludes master nodes from standard load balancer.
	// If not set, it will be default to true.
	ExcludeMasterFromStandardLB *bool `json:"excludeMasterFromStandardLB" yaml:"excludeMasterFromStandardLB"`
	// DisableOutboundSNAT disables the outbound SNAT for public load balancer rules.
	// It should only be set when loadBalancerSku is standard. If not set, it will be default to false.
	DisableOutboundSNAT *bool `json:"disableOutboundSNAT" yaml:"disableOutboundSNAT"`

	// Maximum allowed LoadBalancer Rule Count is the limit enforced by Azure Load balancer
	MaximumLoadBalancerRuleCount int `json:"maximumLoadBalancerRuleCount" yaml:"maximumLoadBalancerRuleCount"`
}

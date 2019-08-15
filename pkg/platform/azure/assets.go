package azure

//CloudProviderConfig is the azure cloud provider config
type CloudProviderConfig struct {
	TenantID       string
	SubscriptionID string
	GroupLocation  string
	ResourcePrefix string
}

// CloudProviderConfig provides the cloud provider config file for Microsoft Azure.
func (a *Azure) CloudProviderConfig(infraID, clusterName string) (string, error) {
	return "", nil
}

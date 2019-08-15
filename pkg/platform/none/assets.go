package none

// CloudProviderConfig provides the cloud provider config file for none.
func (n *None) CloudProviderConfig(infraID, clusterName string) (string, error) {
	return "", nil
}

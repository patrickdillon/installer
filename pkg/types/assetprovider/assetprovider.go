package assetprovider

// AssetProvider defines the functions used to create assets.
type AssetProvider interface {
	Name() string
	ManifestProvider
}

// ManifestProvider specifies the functions to create manifests.
type ManifestProvider interface {
	// CloudProviderConfig returns the data file referenced in the cloud provider configmap.
	// Returning an empty string skips the creation of a cloud provider config.
	//CloudProviderConfig(infraID, clusterName string) (string, error)
}
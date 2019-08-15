package types

import (
	//"context"
	"encoding/json"
	"fmt"

	"github.com/davecgh/go-spew/spew"
	"github.com/openshift/installer/pkg/cloudprovider/aws"
)

// CloudProvider holds the provider interface.
type CloudProvider struct {
	AssetProvider
}

// AssetProvider defines the needed methods to create assets.
type AssetProvider interface {
	Name() string
	ManifestProvider
}

// ManifestProvider specifies the methods providers implement to create manifests.
type ManifestProvider interface {
	// CloudProviderConfig returns the data file referenced in the cloud provider configmap.
	// Returning an empty string skips the creation of a cloud provider config.
	CloudProviderConfig(infraID, clusterName string) (string, error)
}

// Apparently avoids recursion when unmarshalling. check this
type platform CloudProvider

// UnmarshalJSON is a custom unmarshaller used to instantiate a concrete type into the CloudProvider interface.
func (p *CloudProvider) UnmarshalJSON(b []byte) (err error) {
	spew.Dump(b)
	c := make(map[string]interface{})
	json.Unmarshal(b, &c)

	for platform := range c {
		if pi, err := instantiate(platform); err == nil {
			return p.unmarshalProvider(b, pi, platform)
		}
	}
	return fmt.Errorf("error unmarshalling platform from install config")
}

func (p *CloudProvider) unmarshalProvider(b []byte, prov AssetProvider, platform string) error {
	if err := json.Unmarshal(b, &prov); err == nil {
		p.AssetProvider = prov
		spew.Dump(p)
		return nil
	}
	return fmt.Errorf("unable to unmarshal AWS platform in install config")
}

func instantiate(platform string) (AssetProvider, error) {
	
	// In-tree cloud providers.
	platformStructs := map[string]AssetProvider{
		aws.Name:       &aws.AWS{},
	}

	// Out-of-tree provider.	

	if instance, ok := platformStructs[platform]; ok {
		return instance, nil
	}
	return nil, fmt.Errorf("unrecognized platform %q in install-config", platform)

}

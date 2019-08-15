package types

import (
	//"context"
	"encoding/json"
	"fmt"
	"github.com/davecgh/go-spew/spew"
	"github.com/openshift/installer/pkg/platform/aws"
	"github.com/openshift/installer/pkg/platform/azure"
	"github.com/openshift/installer/pkg/platform/baremetal"
	"github.com/openshift/installer/pkg/platform/gcp"
	"github.com/openshift/installer/pkg/platform/libvirt"
	"github.com/openshift/installer/pkg/platform/none"
	"github.com/openshift/installer/pkg/platform/openstack"
	"github.com/openshift/installer/pkg/platform/vsphere"
)

// CloudPlatform holds the provider interface
type CloudPlatform struct {
	Provider
}

// Provider defines the needed methods to create assets
type Provider interface {
	Name() string
	AssetProvider
}

// AssetProvider specifies the methods providers implement to create assets
//
// CloudProviderConfig returns the data file referenced in the cloud provider configmap.
// Returning an empty string skips the creation of a cloud provider config.
type AssetProvider interface {
	CloudProviderConfig(infraID, clusterName string) (string, error)
}

// Apparently avoids recursion when unmarshalling. check this
type platform CloudPlatform

// UnmarshalJSON is a custom unmarshaller used to instantiate a concrete type into the platform interface
func (p *CloudPlatform) UnmarshalJSON(b []byte) (err error) {
	spew.Dump(b)
	c := make(map[string]interface{})
	json.Unmarshal(b, &c)

	// TODO: Currently this is somewhat hacky json.Unmarshal does not seem to throw an error based on the keys
	// for example, platform: gcp can be unmarshaled into "json:aws" without throwing an error
	for platform := range c {
		switch platform {
		case aws.Name:
			return p.unmarshalPlatform(b, &aws.AWS{}, platform)
		case azure.Name:
			return p.unmarshalPlatform(b, &azure.Azure{}, platform)
		case baremetal.Name:
			return p.unmarshalPlatform(b, &baremetal.BareMetal{}, platform)
		case gcp.Name:
			return p.unmarshalPlatform(b, &gcp.GCP{}, platform)
		case libvirt.Name:
			return p.unmarshalPlatform(b, &libvirt.Libvirt{}, platform)
		case none.Name:
			return p.unmarshalPlatform(b, &none.None{}, platform)
		case openstack.Name:
			return p.unmarshalPlatform(b, &openstack.OpenStack{}, platform)
		case vsphere.Name:
			return p.unmarshalPlatform(b, &vsphere.VSphere{}, platform)
		case "default":
			return fmt.Errorf("unrecognized platform %v in install-config", platform)
		}
	}
	return fmt.Errorf("error unmarshalling platform from install config")
}

func (p *CloudPlatform) unmarshalPlatform(b []byte, prov Provider, platform string) error {
	if err := json.Unmarshal(b, &prov); err == nil {
		p.Provider = prov
		return nil
	}
	return fmt.Errorf("unable to unmarshal AWS platform in install config")
}

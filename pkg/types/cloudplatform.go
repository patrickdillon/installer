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
	AssetProvider
}

// Provider defines the needed methods to create assets
type AssetProvider interface {
	Name() string
	ManifestProvider
}

// ManifestProvider specifies the methods providers implement to create assets
//
// CloudProviderConfig returns the data file referenced in the cloud provider configmap.
// Returning an empty string skips the creation of a cloud provider config.
type ManifestProvider interface {
	CloudProviderConfig(infraID, clusterName string) (string, error)
}

// Apparently avoids recursion when unmarshalling. check this
type platform CloudPlatform

// UnmarshalJSON is a custom unmarshaller used to instantiate a concrete type into the platform interface
func (p *CloudPlatform) UnmarshalJSON(b []byte) (err error) {
	spew.Dump(b)
	c := make(map[string]interface{})
	json.Unmarshal(b, &c)

	for platform := range c {
		if pi, err := getInstanceFor(platform); err == nil {
			return p.unmarshalPlatform(b, pi, platform)
		}
	}
	return fmt.Errorf("error unmarshalling platform from install config")
}

func (p *CloudPlatform) unmarshalPlatform(b []byte, prov AssetProvider, platform string) error {
	if err := json.Unmarshal(b, &prov); err == nil {
		p.AssetProvider = prov
		spew.Dump(p)
		return nil
	}
	return fmt.Errorf("unable to unmarshal AWS platform in install config")
}

func getInstanceFor(platform string) (AssetProvider, error) {
	platformStructs := map[string]AssetProvider{
		aws.Name:       &aws.AWS{},
		azure.Name:     &azure.Azure{},
		baremetal.Name: &baremetal.BareMetal{},
		gcp.Name:       &gcp.GCP{},
		libvirt.Name:   &libvirt.Libvirt{},
		none.Name:      &none.None{},
		openstack.Name: &openstack.OpenStack{},
		vsphere.Name:   &vsphere.VSphere{},
	}
	if instance, ok := platformStructs[platform]; ok {
		return instance, nil
	}
	return nil, fmt.Errorf("unrecognized platform %q in install-config", platform)

}

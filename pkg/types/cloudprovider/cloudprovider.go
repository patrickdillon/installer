package cloudprovider

import (
	"encoding/json"
	"fmt"
	"os"
	"plugin"

	"github.com/davecgh/go-spew/spew"
	"github.com/pkg/errors"
	
	"github.com/openshift/installer/pkg/types/assetprovider"
)

const (
	// CloudProviderName provides compatibility with the existing Platform implementation which invokes
	// platform-specific logic based on the platform name. This const indicates the cloud provider
	// interface should be used.
	CloudProviderName = "CLOUD_PROVIDER_PLUGIN"

	// pluginLoadFunc is the name of the function for loading
	// the platform from the plugin.
	pluginLoadFunc = "Load"
)

// CloudProvider implements the AssetProvider interface.
// Embedding the DefaultProvider provides a method of
// backward compatibility for plugins.
type CloudProvider struct {
	assetprovider.AssetProvider
	DefaultProvider
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

func (p *CloudProvider) unmarshalProvider(b []byte, prov assetprovider.AssetProvider, platform string) error {
	if err := json.Unmarshal(b, &prov); err == nil {
		p.AssetProvider = prov
		return nil
	}
	return fmt.Errorf("unable to unmarshal platform plugin in install config")
}

func instantiate(platform string) (assetprovider.AssetProvider, error) {

	// In-tree cloud providers.
	platformStructs := map[string]assetprovider.AssetProvider{
		// In-tree providers could be added here. e.g.
		// aws.Name: &aws.AWS{},
	}

	// Out-of-tree provider.
	err := loadPlugins(platformStructs)
	if err != nil {
		return nil, errors.Wrap(err, "error loading plugin")
	}

	if instance, ok := platformStructs[platform]; ok {
		return instance, nil
	}
	return nil, fmt.Errorf("unrecognized platform %q in install-config", platform)
}

func loadPlugins(platforms map[string]assetprovider.AssetProvider) error {
	fmt.Println("DEBUG START PLUGIN loadplugins")

	// TODO: evaluate UX other than envvar for passing plugin path, e,g. cli flag
	if pp := os.Getenv("OPENSHIFT_INSTALL_PLUGIN_PATH"); pp != "" {
		fmt.Println("found envvar")
		p, err := plugin.Open(pp)
		if err != nil {
			fmt.Printf("error opening cloud-provider plugin %s \n", pp)
			fmt.Println(err.Error())
			return errors.Wrapf(err, "error opening cloud-provider plugin at %q", pp)
		}
		l, err := p.Lookup(pluginLoadFunc)
		if err != nil {
			fmt.Println("error retrieving symbol")
			return errors.Wrapf(err, "error retrieving symbol %s from plugin at %q", pluginLoadFunc, pp)
		}
		fmt.Println("About to load it up")
		load := l.(func(map[string]assetprovider.AssetProvider) (string, error))
		load(platforms) // TODO error checking, assert type
		fmt.Println("Done loading it up!")
		fmt.Println(platforms)
	}
	fmt.Println("DEBUG returning")
	return nil
}

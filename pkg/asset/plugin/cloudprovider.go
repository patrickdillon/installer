package cloudproviderplugin

import (
	"os"
	"plugin"

	"github.com/pkg/errors"

	"github.com/openshift/installer/pkg/asset"
	"github.com/openshift/installer/pkg/types"
)

const cpSym = "CloudProvider"

// cloudproviderplugin
type CloudProviderPlugin types.CloudProvider

var _ asset.Asset = (*CloudProviderPlugin)(nil)

// Dependencies returns no dependencies.
func (a *CloudProviderPlugin) Dependencies() []asset.Asset {
	return []asset.Asset{}
}

// Generate loads any needed plugins.
func (a *CloudProviderPlugin) Generate(asset.Parents) error {
	// TODO: evaluate UX for passing plugin path, e,g. cli flag
	if pp := os.Getenv("OPENSHIFT_INSTALL_PLUGIN_PATH"); pp != "" {
		p, err := plugin.Open(pp)
		if err != nil {
			return errors.Wrapf(err, "error opening cloud-provider plugin at %q", pp)
		}
		cpPlugin, err := p.Lookup(cpSym)
		if err != nil {
			return errors.Wrapf(err, "error retrieving symbol %s from plugin at %q", cpSym, pp)
		}
		if cp, ok := cpPlugin.(types.CloudProvider); ok {
			*a = CloudProviderPlugin(cp)
		}
	}
	return nil
}

// Name returns the human-friendly name of the asset.
func (a *CloudProviderPlugin) Name() string {
	return "Cloud Provider Plugin"
}

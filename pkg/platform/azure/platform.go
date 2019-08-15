package azure

import (
	"github.com/openshift/installer/pkg/types/azure"
)

// Azure holds specifics for the Amazon web services platform.
type Azure struct {
	azure.Platform `json:"azure"`
}

// Name specifies that the platform is azure
func (a *Azure) Name() string {
	return Name
}

package vsphere

import (
	"github.com/openshift/installer/pkg/types/vsphere"
)

// VSphere holds specifics for the VSphere platform.
type VSphere struct {
	vsphere.Platform `json:"VSphere"`
}

// Name returns a constant string literal of the clustername, i.e. "vsphere"
func (v *VSphere) Name() string {
	return Name
}

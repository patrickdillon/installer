package libvirt

import (
	"github.com/openshift/installer/pkg/types/libvirt"
)

// Libvirt holds specifics for the Amazon web services platform.
type Libvirt struct {
	libvirt.Platform `json:"libvirt"`
}

// Name returns a constant string literal of the clustername, i.e. "libvirt"
func (l *Libvirt) Name() string {
	return Name
}

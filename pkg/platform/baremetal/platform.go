package baremetal

import (
	"github.com/openshift/installer/pkg/types/baremetal"
)

// BareMetal holds specifics for the Amazon web services platform.
type BareMetal struct {
	baremetal.Platform `json:"baremetal"`
}

// Name returns a constant string literal of the clustername, i.e. "baremetal"
func (b *BareMetal) Name() string {
	return Name
}

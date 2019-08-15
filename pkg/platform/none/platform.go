package none

import (
	"github.com/openshift/installer/pkg/types/none"
)

// None holds specifics for the none platform.
type None struct {
	none.Platform `json:"None"`
}

// Name returns a constant string literal of the clustername, i.e. "none"
func (n *None) Name() string {
	return Name
}

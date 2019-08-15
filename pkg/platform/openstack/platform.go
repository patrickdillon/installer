package openstack

import (
	"github.com/openshift/installer/pkg/types/openstack"
)

// OpenStack holds specifics for the OpenStack platform.
type OpenStack struct {
	openstack.Platform `json:"OpenStack"`
}

// Name returns a constant string literal of the clustername, i.e. "openstack"
func (o *OpenStack) Name() string {
	return Name
}

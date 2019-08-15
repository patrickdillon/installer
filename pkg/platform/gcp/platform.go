package gcp

import (
	"github.com/openshift/installer/pkg/types/gcp"
)

// GCP holds specifics for the Google cloud platform.
type GCP struct {
	gcp.Platform `json:"gcp"`
}

// Name returns a constant string literal of the clustername, i.e. "gcp"
func (g *GCP) Name() string {
	return Name
}

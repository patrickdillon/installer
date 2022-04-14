package providers

import (
	"github.com/sirupsen/logrus"

	"github.com/openshift/installer/pkg/types"
)

// Gather allows multiple implementations of gather
// for different platforms.
type Gather interface {
	Run() error
}

// NewFunc is an interface for creating platform-specific gather methods.
type NewFunc func(logger logrus.FieldLogger, serialLogBundle, bootstrapIP, bootstrapID string, masterIPs, masterIDs []string, metadata *types.ClusterMetadata) (Gather, error)

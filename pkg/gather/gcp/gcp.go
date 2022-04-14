package gcp

import (
	"github.com/sirupsen/logrus"

	"github.com/openshift/installer/pkg/gather/providers"
	"github.com/openshift/installer/pkg/types"
	//"github.com/openshift/installer/pkg/types/gcp"
)

type Gather struct {
}

func New(logger logrus.FieldLogger, serialLogBundle, bootstrapIP, bootstrapID string, masterIPs, masterIDs []string, metadata *types.ClusterMetadata) (providers.Gather, error) {
	return &Gather{}, nil
}

func (g *Gather) Run() error {
	return nil
}

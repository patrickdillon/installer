package aws

import (
	"github.com/sirupsen/logrus"

	"github.com/openshift/installer/pkg/gather/providers"
	"github.com/openshift/installer/pkg/types"
	//"github.com/openshift/installer/pkg/types/aws"
)

type Gather struct {
}

func New(logger logrus.FieldLogger, serialLogBundle string, bootstrap string, masters []string, metadata *types.ClusterMetadata) (providers.Gather, error) {
	return &Gather{}, nil
}

func (g *Gather) Run() error {
	return nil
}

package aws

import (
	"github.com/openshift/installer/pkg/types/aws"
)

// AWS holds specifics for the Amazon web services platform.
type AWS struct {
	aws.Platform `json:"aws"`
}

// Name specifies that the platform is AWS
func (a *AWS) Name() string {
	return Name
}

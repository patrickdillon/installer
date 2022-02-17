package cloudprovider

import (
	"github.com/sirupsen/logrus"
)

type DefaultProvider struct{}

func warn(funcName string) {
	logrus.Warnf("Function %s not implemented. Returning zero values.", funcName)
}

func (d DefaultProvider) CloudProviderConfig(infraID, clusterName string) (string, error) {
	warn("CloudProviderConfig")
	return "", nil
}

func (d DefaultProvider) Name() string{
	warn("Name")
	return ""
}
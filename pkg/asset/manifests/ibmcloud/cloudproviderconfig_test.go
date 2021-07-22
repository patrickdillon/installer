package ibmcloud

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestCloudProviderConfig(t *testing.T) {
	expectedConfig := `[global]
version = 1.1.0
[kubernetes]
config-file = ""
[provider]
accountID = 1e1f75646aef447814a6d907cc83fb3c
clusterID = ocp4-8pxks
cluster-default-provider = g2
g2workerServiceAccountID = 1e1f75646aef447814a6d907cc83fb3c

`

	actualConfig, err := CloudProviderConfig("ocp4-8pxks", "1e1f75646aef447814a6d907cc83fb3c")
	assert.NoError(t, err, "failed to create cloud provider config")
	assert.Equal(t, expectedConfig, actualConfig, "unexpected cloud provider config")
}

package aws

import (
	"context"
	"time"

	"sigs.k8s.io/cluster-api/cmd/clusterctl/client"
	logf "sigs.k8s.io/cluster-api/cmd/clusterctl/log"
)

func InitializeProvider(kubeconfig string) error {
	ctx := context.Background()

	c, err := client.New(ctx, "")
	if err != nil {
		return err
	}

	options := client.InitOptions{
		// FIX: make kubeconfig usage consistent
		Kubeconfig:              client.Kubeconfig{Path: kubeconfig},
		InfrastructureProviders: []string{"aws"},
		LogUsageInstructions:    true,
		WaitProviders:           false,
		WaitProviderTimeout:     time.Duration(300) * time.Second,
	}

	nl := logf.NewLogger()
	logf.SetLogger(nl)

	if _, err := c.Init(ctx, options); err != nil {
		return err
	}
	return nil
}

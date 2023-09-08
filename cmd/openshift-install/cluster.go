package main

import (
	"github.com/spf13/cobra"

	targetassets "github.com/openshift/installer/pkg/asset/targets"
)

func newCAPIClusterCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "capi-cluster",
		Short: "Create part of an OpenShift cluster",
		Run:   runTargetCmd(targetassets.CAPICluster...),
		Args:  cobra.ExactArgs(0),
	}

	return cmd
}

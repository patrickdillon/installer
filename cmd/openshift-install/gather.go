package main

import (
	"archive/tar"
	"compress/gzip"
	"context"
	"fmt"
	"io"
	"io/ioutil"
	"log"
	"net"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"syscall"
	"time"

	configv1 "github.com/openshift/api/config/v1"
	configclient "github.com/openshift/client-go/config/clientset/versioned"
	"github.com/pkg/errors"
	"github.com/sirupsen/logrus"
	"github.com/spf13/cobra"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/rest"

	"github.com/openshift/installer/pkg/asset/installconfig"
	assetstore "github.com/openshift/installer/pkg/asset/store"
	"github.com/openshift/installer/pkg/asset/tls"
	"github.com/openshift/installer/pkg/gather"
	_ "github.com/openshift/installer/pkg/gather/aws"
	_ "github.com/openshift/installer/pkg/gather/azure"
	_ "github.com/openshift/installer/pkg/gather/gcp"
	"github.com/openshift/installer/pkg/gather/service"
	"github.com/openshift/installer/pkg/gather/ssh"
	platformstages "github.com/openshift/installer/pkg/terraform/stages/platform"
)

func newGatherCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "gather",
		Short: "Gather debugging data for a given installation failure",
		Long: `Gather debugging data for a given installation failure.

When installation for OpenShift cluster fails, gathering all the data useful for debugging can
become a difficult task. This command helps users to collect the most relevant information that can be used
to debug the installation failures`,
		RunE: func(cmd *cobra.Command, args []string) error {
			return cmd.Help()
		},
	}
	cmd.AddCommand(newGatherBootstrapCmd())
	return cmd
}

var (
	gatherBootstrapOpts struct {
		bootstrap    string
		masters      []string
		sshKeys      []string
		skipAnalysis bool
	}
)

func newGatherBootstrapCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "bootstrap",
		Short: "Gather debugging data for a failing-to-bootstrap control plane",
		Args:  cobra.ExactArgs(0),
		Run: func(_ *cobra.Command, _ []string) {
			cleanup := setupFileHook(rootOpts.dir)
			defer cleanup()
			serialBundlePath, bundlePath, err := runGatherBootstrapCmd(rootOpts.dir)
			if err != nil {
				logrus.Fatal(err)
			}

			if !gatherBootstrapOpts.skipAnalysis {
				if err := service.AnalyzeGatherBundle(bundlePath); err != nil {
					logrus.Fatal(err)
				}
			}

			logrus.Infof("Serial Bootstrap gather logs captured here %q", serialBundlePath)
			logrus.Infof("Bootstrap gather logs captured here %q", bundlePath)
		},
	}
	cmd.PersistentFlags().StringVar(&gatherBootstrapOpts.bootstrap, "bootstrap", "", "Hostname or IP of the bootstrap host")
	cmd.PersistentFlags().StringArrayVar(&gatherBootstrapOpts.masters, "master", []string{}, "Hostnames or IPs of all control plane hosts")
	cmd.PersistentFlags().StringArrayVar(&gatherBootstrapOpts.sshKeys, "key", []string{}, "Path to SSH private keys that should be used for authentication. If no key was provided, SSH private keys from user's environment will be used")
	cmd.PersistentFlags().BoolVar(&gatherBootstrapOpts.skipAnalysis, "skipAnalysis", false, "Skip analysis of the gathered data")
	return cmd
}

func runGatherBootstrapCmd(directory string) (string, string, error) {
	assetStore, err := assetstore.NewStore(directory)
	if err != nil {
		return "", "", errors.Wrap(err, "failed to create asset store")
	}
	// add the default bootstrapIP key pair to the sshKeys list
	bootstrapSSHKeyPair := &tls.BootstrapSSHKeyPair{}
	if err := assetStore.Fetch(bootstrapSSHKeyPair); err != nil {
		return "", "", errors.Wrapf(err, "failed to fetch %s", bootstrapSSHKeyPair.Name())
	}
	tmpfile, err := ioutil.TempFile("", "bootstrapIP-ssh")
	if err != nil {
		return "", "", err
	}
	defer os.Remove(tmpfile.Name())
	if _, err := tmpfile.Write(bootstrapSSHKeyPair.Private()); err != nil {
		return "", "", err
	}
	if err := tmpfile.Close(); err != nil {
		return "", "", err
	}
	gatherBootstrapOpts.sshKeys = append(gatherBootstrapOpts.sshKeys, tmpfile.Name())

	bootstrapIP := gatherBootstrapOpts.bootstrap
	port := 22
	masterIPs := gatherBootstrapOpts.masters
	var bootstrapID string
	var masterIDs []string
	if bootstrapIP == "" && len(masterIPs) == 0 {
		config := &installconfig.InstallConfig{}
		if err := assetStore.Fetch(config); err != nil {
			return "", "", errors.Wrapf(err, "failed to fetch %s", config.Name())
		}

		for _, stage := range platformstages.StagesForPlatform(config.Config.Platform.Name()) {
			stageBootstrapIP, stagePort, stageMasterIPs, err := stage.ExtractHostAddresses(directory, config.Config)
			if err != nil {
				return "", "", err
			}
			if stageBootstrapIP != "" {
				bootstrapIP = stageBootstrapIP
			}
			if stagePort != 0 {
				port = stagePort
			}
			if len(stageMasterIPs) > 0 {
				masterIPs = stageMasterIPs
			}

			//TODO: Seems like this logic could be refactored into a closure
			stageBootstrapID, stageMasterIDs, err := stage.ExtractHostIDs(directory, config.Config)
			if stageBootstrapID != "" {
				bootstrapID = stageBootstrapID
			}
			if len(stageMasterIDs) > 0 {
				masterIDs = stageMasterIDs
			}
		}

		if bootstrapIP == "" || len(masterIPs) == 0 {
			return "", "", errors.New("bootstrap host address and at least one control plane host address must be provided")
		}
	} else if bootstrapIP == "" || len(masterIPs) == 0 {
		return "", "", errors.New("must provide both bootstrap host address and at least one control plane host address when providing one")
	}

	return gatherBootstrap(bootstrapIP, bootstrapID, port, masterIPs, masterIDs, directory)
}

func gatherBootstrap(bootstrapIP, bootstrapID string, port int, masterIPs, masterIDs []string, directory string) (string, string, error) {
	gatherID := time.Now().Format("20060102150405")

	serialLogBundle := filepath.Join(directory, fmt.Sprintf("serial-log-bundle-%s.tar.gz", gatherID))
	serialLogBundlePath, err := filepath.Abs(serialLogBundle)

	logrus.Info("Pulling platform specific console logs from the bootstrap machine")
	gather, err := gather.New(logrus.StandardLogger(), serialLogBundle, bootstrapIP, bootstrapID, masterIPs, masterIDs, directory)
	if err != nil {
		return "", "", errors.Wrap(err, "Failed to create gather object")
	}
	err = gather.Run()
	if err != nil {
		logrus.Infof("%s: Failed to gather platform specific logs", err.Error())
	}

	logrus.Info("Pulling debug logs from the bootstrap machine")
	client, err := ssh.NewClient("core", net.JoinHostPort(bootstrapIP, strconv.Itoa(port)), gatherBootstrapOpts.sshKeys)
	if err != nil {
		if errors.Is(err, syscall.ECONNREFUSED) || errors.Is(err, syscall.ETIMEDOUT) {
			return "", "", errors.Wrap(err, "failed to connect to the bootstrapIP machine")
		}
		return "", "", errors.Wrap(err, "failed to create SSH client")
	}

	if err := ssh.Run(client, fmt.Sprintf("/usr/local/bin/installer-gather.sh --id %s %s", gatherID, strings.Join(masterIPs, " "))); err != nil {
		return "", "", errors.Wrap(err, "failed to run remote command")
	}
	file := filepath.Join(directory, fmt.Sprintf("log-bundle-%s.tar.gz", gatherID))
	if err := ssh.PullFileTo(client, fmt.Sprintf("/home/core/log-bundle-%s.tar.gz", gatherID), file); err != nil {
		return "", "", errors.Wrap(err, "failed to pull log file from remote")
	}
	logBundlePath, err := filepath.Abs(file)
	if err != nil {
		return "", "", errors.Wrap(err, "failed to stat log file")
	}

	combinedLogBundlePath, err := appendLogs(logBundlePath, serialLogBundle, directory, gatherID)
	if err != nil {
		return "", "", errors.Wrap(err, "failed to combine log bundles")
	}
	return serialLogBundlePath, combinedLogBundlePath, nil
}

func logClusterOperatorConditions(ctx context.Context, config *rest.Config) error {
	client, err := configclient.NewForConfig(config)
	if err != nil {
		return errors.Wrap(err, "creating a config client")
	}

	operators, err := client.ConfigV1().ClusterOperators().List(ctx, metav1.ListOptions{})
	if err != nil {
		return errors.Wrap(err, "listing ClusterOperator objects")
	}

	for _, operator := range operators.Items {
		for _, condition := range operator.Status.Conditions {
			if condition.Type == configv1.OperatorUpgradeable {
				continue
			} else if condition.Type == configv1.OperatorAvailable && condition.Status == configv1.ConditionTrue {
				continue
			} else if (condition.Type == configv1.OperatorDegraded || condition.Type == configv1.OperatorProgressing) && condition.Status == configv1.ConditionFalse {
				continue
			}
			if condition.Type == configv1.OperatorDegraded {
				logrus.Errorf("Cluster operator %s %s is %s with %s: %s", operator.ObjectMeta.Name, condition.Type, condition.Status, condition.Reason, condition.Message)
			} else {
				logrus.Infof("Cluster operator %s %s is %s with %s: %s", operator.ObjectMeta.Name, condition.Type, condition.Status, condition.Reason, condition.Message)
			}
		}
	}

	return nil
}

// append adds the serial bundle to the end of the gather bundle in a single gzipped archive.
func appendLogs(gb, sb, dir, gatherID string) (string, error) {
	logrus.Info("Combining serial and bootstrap log bundles")
	file := filepath.Join(dir, fmt.Sprintf("combined-log-bundle-%s.tar.gz", gatherID))
	bundlePath, err := filepath.Abs(file)
	zf, err := os.Create(bundlePath)
	if err != nil {
		return "", err
	}
	zw := gzip.NewWriter(zf)
	tw := tar.NewWriter(zw)

	for _, v := range []string{gb, sb} {
		fmt.Printf("===Opening %s=== \n", v)
		f, err := os.Open(v)
		if err != nil {
			return "", errors.Wrapf(err, "error opening %s to combine log bundles", v)
		}

		zr, err := gzip.NewReader(f)
		if err != nil {
			return "", errors.Wrapf(err, "error creating gzip reader for %s", v)
		}

		tr := tar.NewReader(zr)
		for {
			hdr, err := tr.Next()
			if err == io.EOF {
				break // End of archive
			}
			if err != nil {
				return "", errors.Wrapf(err, "error reading contents of %s", v)
			}

			//TODO hack
			if v == sb {
				tmpHdr := strings.SplitN(hdr.Name, "/", 3)[2]
				hdr.Name = fmt.Sprintf("log-bundle-%s/serial-logs/%s", gatherID, tmpHdr)
			}
			if err := tw.WriteHeader(hdr); err != nil {
				log.Fatal(err)
			}
			if _, err := io.Copy(tw, tr); err != nil {
				log.Fatal(err)
			}
			fmt.Println(hdr)
		}
	}

	if err := tw.Close(); err != nil {
		return "", errors.Wrap(err, "error closing tar writer")
	}
	if err := zw.Close(); err != nil {
		return "", errors.Wrap(err, "error closing zip writer")
	}
	return bundlePath, nil
}

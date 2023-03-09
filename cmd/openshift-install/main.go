package main

import (
	"flag"
	"io"
	"os"
	"path/filepath"
	"sort"

	"github.com/pkg/errors"
	"github.com/sirupsen/logrus"
	"github.com/spf13/cobra"
	terminal "golang.org/x/term"
	"k8s.io/klog"
	klogv2 "k8s.io/klog/v2"

	configv1 "github.com/openshift/api/config/v1"
)

var (
	rootOpts struct {
		dir        string
		logLevel   string
		featureSet string
	}
)

func main() {
	// This attempts to configure klog (used by vendored Kubernetes code) not
	// to log anything.
	// Handle k8s.io/klog
	var fs flag.FlagSet
	klog.InitFlags(&fs)
	fs.Set("stderrthreshold", "4")
	klog.SetOutput(io.Discard)
	// Handle k8s.io/klog/v2
	var fsv2 flag.FlagSet
	klogv2.InitFlags(&fsv2)
	fsv2.Set("stderrthreshold", "4")
	klogv2.SetOutput(io.Discard)

	installerMain()
}

func installerMain() {
	rootCmd := newRootCmd()

	for _, subCmd := range []*cobra.Command{
		newCreateCmd(),
		newDestroyCmd(),
		newWaitForCmd(),
		newGatherCmd(),
		newAnalyzeCmd(),
		newVersionCmd(),
		newGraphCmd(),
		newCoreOSCmd(),
		newCompletionCmd(),
		newMigrateCmd(),
		newExplainCmd(),
		newAgentCmd(),
	} {
		rootCmd.AddCommand(subCmd)
	}

	if err := rootCmd.Execute(); err != nil {
		logrus.Fatalf("Error executing openshift-install: %v", err)
	}
}

func newRootCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:              filepath.Base(os.Args[0]),
		Short:            "Creates OpenShift clusters",
		Long:             "",
		PersistentPreRun: runRootCmd,
		SilenceErrors:    true,
		SilenceUsage:     true,
	}
	cmd.PersistentFlags().StringVar(&rootOpts.dir, "dir", ".", "assets directory")
	cmd.PersistentFlags().StringVar(&rootOpts.logLevel, "log-level", "info", "log level (e.g. \"debug | info | warn | error\")")
	cmd.PersistentFlags().StringVar(&rootOpts.featureSet, "feature-set", "", "feature set to enable during install and on the installed cluster")
	return cmd
}

func runRootCmd(cmd *cobra.Command, args []string) {
	setupLogging()
	validateFeatureSet()
}

func setupLogging() {
	logrus.SetOutput(io.Discard)
	logrus.SetLevel(logrus.TraceLevel)

	level, err := logrus.ParseLevel(rootOpts.logLevel)
	if err != nil {
		level = logrus.InfoLevel
	}

	logrus.AddHook(newFileHookWithNewlineTruncate(os.Stderr, level, &logrus.TextFormatter{
		// Setting ForceColors is necessary because logrus.TextFormatter determines
		// whether or not to enable colors by looking at the output of the logger.
		// In this case, the output is io.Discard, which is not a terminal.
		// Overriding it here allows the same check to be done, but against the
		// hook's output instead of the logger's output.
		ForceColors:            terminal.IsTerminal(int(os.Stderr.Fd())),
		DisableTimestamp:       true,
		DisableLevelTruncation: true,
		DisableQuote:           true,
	}))

	if err != nil {
		logrus.Fatal(errors.Wrap(err, "Invalid log-level"))
	}
}

func validateFeatureSet() {
	fs := configv1.FeatureSet(rootOpts.featureSet)
	if _, ok := configv1.FeatureSets[fs]; !ok {
		sortedFeatureSets := func() []string {
			v := []string{}
			for n := range configv1.FeatureSets {
				v = append(v, string(n))
			}
			sort.Strings(v)
			return v
		}()
		logrus.Fatalf("Invalid --feature-set flag %q: supported values: %q", fs, sortedFeatureSets)
	}
}

// gateOnTechPreviewNoUpgrade is a helper function that can be added
// (e.g. as a PreRun command) to an existing cobra command to prevent it
// from running unless --feature-set TechPreviewNoUpgrade is set.
func gateCmdOn(fs configv1.FeatureSet) func(*cobra.Command, []string) {
	return func(cmd *cobra.Command, _ []string) {
		if configv1.FeatureSet(rootOpts.featureSet) != fs {
			//TO DO: Line break and link to feature set docs
			logrus.Fatalf("The %q command is in a non-default feature set: to opt in use the flag --feature-set %s", cmd.Use, fs)
		}
	}
}

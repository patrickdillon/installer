package azure

import (
	"context"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/Azure/azure-sdk-for-go/sdk/azcore/arm"
	"github.com/Azure/azure-sdk-for-go/sdk/azcore/policy"
	"github.com/Azure/azure-sdk-for-go/sdk/azcore/to"
	"github.com/Azure/azure-sdk-for-go/sdk/resourcemanager/compute/armcompute/v4"
	"github.com/pkg/errors"
	"github.com/sirupsen/logrus"
	utilerrors "k8s.io/apimachinery/pkg/util/errors"

	azuresession "github.com/openshift/installer/pkg/asset/installconfig/azure"
	"github.com/openshift/installer/pkg/gather"
	"github.com/openshift/installer/pkg/gather/providers"
	"github.com/openshift/installer/pkg/types"
	"github.com/openshift/installer/pkg/types/azure"
)

// Gather holds options for resources we want to gather.
type Gather struct {
	resourceGroupName     string
	logger                logrus.FieldLogger
	serialLogBundle       string
	directory             string
	virtualMachinesClient *armcompute.VirtualMachinesClient
	session               *azuresession.Session
}

// New returns a Azure Gather from ClusterMetadata.
func New(logger logrus.FieldLogger, serialLogBundle string, bootstrap string, masters []string, metadata *types.ClusterMetadata) (providers.Gather, error) {
	cloudName := metadata.Azure.CloudName
	if cloudName == "" {
		cloudName = azure.PublicCloud
	}

	resourceGroupName := metadata.Azure.ResourceGroupName
	if resourceGroupName == "" {
		resourceGroupName = metadata.InfraID + "-rg"
	}

	session, err := azuresession.GetSession(cloudName, metadata.Azure.ARMEndpoint)
	if err != nil {
		return nil, err
	}

	options := arm.ClientOptions{
		ClientOptions: policy.ClientOptions{
			Cloud: session.CloudConfig,
		},
	}
	virtualMachinesClient, err := armcompute.NewVirtualMachinesClient(session.Credentials.SubscriptionID, session.TokenCreds, &options)
	if err != nil {
		return nil, err
	}

	gather := &Gather{
		resourceGroupName:     resourceGroupName,
		logger:                logger,
		serialLogBundle:       serialLogBundle,
		directory:             filepath.Dir(serialLogBundle),
		virtualMachinesClient: virtualMachinesClient,
		session:               session,
	}

	return gather, nil
}

// Run is the entrypoint to start the gather process.
func (g *Gather) Run() error {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Minute)
	defer cancel()

	virtualMachines, err := g.getVirtualMachines(ctx)
	if err != nil {
		return err
	}

	// We can only get the serial log from VM's with boot diagnostics enabled
	files := g.getBootDiagnostics(ctx, virtualMachines)
	if len(files) > 0 {
		err = g.downloadFiles(ctx, files)
		if err != nil {
			return err
		}
	} else {
		g.logger.Debugln("No boot diagnostics found")
	}

	return nil
}

func (g *Gather) getVirtualMachines(ctx context.Context) ([]*armcompute.VirtualMachine, error) {
	var virtualMachines []*armcompute.VirtualMachine
	vmsPager := g.virtualMachinesClient.NewListPager(g.resourceGroupName, nil)
	for vmsPager.More() {
		vmPage, err := vmsPager.NextPage(ctx)
		if err != nil {
			g.logger.Debugf("failed to get vm page: %v", err)
			continue
		}
		virtualMachines = append(virtualMachines, vmPage.Value...)
	}
	return virtualMachines, nil
}

func (g *Gather) getBootDiagnostics(ctx context.Context, virtualMachines []*armcompute.VirtualMachine) []string {
	var bootDiagnostics []string
	options := armcompute.VirtualMachinesClientRetrieveBootDiagnosticsDataOptions{
		SasURIExpirationTimeInMinutes: to.Ptr[int32](60),
	}
	for _, vm := range virtualMachines {
		logger := g.logger.WithField("VM", *vm.Name)
		if vm.Properties == nil || vm.Properties.DiagnosticsProfile == nil || vm.Properties.DiagnosticsProfile.BootDiagnostics == nil {
			logger.Debug("no boot diagnostics found for VM")
			continue
		}
		if vm.Properties.DiagnosticsProfile.BootDiagnostics.Enabled == nil || !*vm.Properties.DiagnosticsProfile.BootDiagnostics.Enabled {
			logger.Debug("boot diagnostics are not enabled for VM, skipping")
			continue
		}
		res, err := g.virtualMachinesClient.RetrieveBootDiagnosticsData(ctx, g.resourceGroupName, *vm.Name, &options)
		if err != nil {
			logger.Debugf("failed to get boot diagnostics for VM: %v", err)
			continue
		}
		if res.ConsoleScreenshotBlobURI != nil {
			bootDiagnostics = append(bootDiagnostics, *res.ConsoleScreenshotBlobURI)
		}
		if res.SerialConsoleLogBlobURI != nil {
			bootDiagnostics = append(bootDiagnostics, *res.SerialConsoleLogBlobURI)
		}
	}

	return bootDiagnostics
}

func (g *Gather) downloadFiles(ctx context.Context, fileURIs []string) error {
	var errs []error
	var files []string

	serialLogBundleDir := filepath.Join(g.directory, strings.TrimSuffix(filepath.Base(g.serialLogBundle), ".tar.gz"))
	err := os.MkdirAll(serialLogBundleDir, 0o755)
	if err != nil && !errors.Is(err, os.ErrExist) {
		return err
	}

	for _, fileURI := range fileURIs {
		logger := g.logger.WithField("fileURI", fileURI)
		filePath, ferr := downloadFile(ctx, fileURI, serialLogBundleDir, logger)
		if ferr != nil {
			errs = append(errs, ferr)
			continue
		}
		files = append(files, filePath)
	}

	if len(files) > 0 {
		err = gather.CreateArchive(files, g.serialLogBundle)
		if err != nil {
			g.logger.Debugf("failed to create archive: %s", err.Error())
			errs = append(errs, err)
		}
	}

	err = gather.DeleteArchiveDirectory(serialLogBundleDir)
	if err != nil {
		g.logger.Debugf("failed to remove archive directory: %v", err)
	}

	return utilerrors.NewAggregate(errs)
}

func downloadFile(ctx context.Context, fileURI string, filePathDir string, logger logrus.FieldLogger) (string, error) {
	logger.Debugln("attemping to download file")

	// Remove any possible token from the URI
	fileName, _, _ := strings.Cut(fileURI, "?")
	filePath := filepath.Join(filePathDir, filepath.Base(fileName))

	file, err := os.Create(filePath)
	if err != nil {
		logger.Debugf("failed to create file \"%s\": %s", filePath, err.Error())
		return "", err
	}
	defer file.Close()

	req, err := http.NewRequestWithContext(ctx, "GET", fileURI, nil)
	if err != nil {
		return "", err
	}
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return "", errors.Errorf("unable to download file: %s", filePath)
	}

	_, err = io.Copy(file, resp.Body)
	if err != nil {
		logger.Debugf("failed to write to file: %s", err.Error())
		return "", err
	}

	return filePath, nil
}

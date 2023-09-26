package controlplane

import (
	"os"

	"github.com/davecgh/go-spew/spew"
	"github.com/pkg/errors"
	"k8s.io/client-go/kubernetes"
	"sigs.k8s.io/controller-runtime/pkg/envtest"
)

var (
	// specify testEnv configuration

	// copy of default flags, remove insecure port and add service account stuff
	flags = []string{
		"--advertise-address=127.0.0.1",
		"--etcd-servers={{ if .EtcdURL }}{{ .EtcdURL.String }}{{ end }}",
		"--cert-dir={{ .CertDir }}",
		//"--insecure-port={{ if .URL }}{{ .URL.Port }}{{else}}0{{ end }}",
		"{{ if .URL }}--insecure-bind-address={{ .URL.Hostname }}{{ end }}",
		"--secure-port={{ if .SecurePort }}{{ .SecurePort }}{{ end }}",
		// we're keeping this disabled because if enabled, default SA is missing which would force all tests to create one
		// in normal apiserver operation this SA is created by controller, but that is not run in integration environment
		"--disable-admission-plugins=ServiceAccount",
		"--service-cluster-ip-range=10.0.0.0/24",
		"--allow-privileged=true",
		"--service-account-key-file=/home/padillon/hacking/service-account.pem",
		"--service-account-signing-key-file=/home/padillon/hacking/service-account-key.pem",
		"--service-account-issuer=https",
	}

	localCAPIEnv = &envtest.Environment{
		//CRDDirectoryPaths: []string{filepath.Join("..", "config", "crd", "bases")},
		AttachControlPlaneOutput: true,
		KubeAPIServerFlags:       flags,
	}
)

func Start(installDir string) ([]byte, error) {

	// TODO: This is taking external binaries. We will want to
	// embed these binaries like we do terraform resources.
	etcd := os.Getenv("OPENSHIFT_INSTALL_ETCD")
	api := os.Getenv("OPENSHIFT_INSTALL_API")
	kubectl := os.Getenv("OPENSHIFT_INSTALL_KUBECTL")

	if err := os.Setenv("TEST_ASSET_KUBE_APISERVER", api); err != nil {
		return nil, err
	}
	if err := os.Setenv("TEST_ASSET_ETCD", etcd); err != nil {
		return nil, err
	}
	if err := os.Setenv("TEST_ASSET_KUBECTL", kubectl); err != nil {
		return nil, err
	}

	cfg, err := localCAPIEnv.Start()
	spew.Dump("===cfg===")
	spew.Dump(cfg)

	usr := envtest.User{Name: "myuser"}
	authUsr, err := localCAPIEnv.AddUser(usr, cfg)
	if err != nil {
		return nil, err
	}

	kc, err := authUsr.KubeConfig()
	if err != nil {
		return nil, err
	}

	client, err := kubernetes.NewForConfig(cfg)
	if err != nil {
		return nil, errors.Wrap(err, "creating a Kubernetes client")
	}

	discovery := client.Discovery()
	v, _ := discovery.ServerVersion()
	spew.Dump(v)

	// localCAPIEnv.Stop()
	// panic("stopping here")
	return kc, err
}

func Stop() error {
	return localCAPIEnv.Stop()
}

package machineconfig

import (
	"fmt"

	igntypes "github.com/coreos/ignition/v2/config/v3_2/types"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	mcfgv1 "github.com/openshift/api/machineconfiguration/v1"
	"github.com/openshift/installer/pkg/asset/ignition"
)

// ForCRIODefaultEnv creates a MachineConfig that drops a CRI-O config
// snippet injecting the given environment variables into every container.
func ForCRIODefaultEnv(role string, envVars map[string]string) (*mcfgv1.MachineConfig, error) {
	entries := ""
	i := 0
	for k, v := range envVars {
		if i > 0 {
			entries += ", "
		}
		entries += fmt.Sprintf("%q", k+"="+v)
		i++
	}
	conf := fmt.Sprintf("[crio.runtime]\ndefault_env = [%s]\n", entries)

	ignConfig := igntypes.Config{
		Ignition: igntypes.Ignition{
			Version: igntypes.MaxVersion.String(),
		},
		Storage: igntypes.Storage{
			Files: []igntypes.File{
				ignition.FileFromString("/etc/crio/crio.conf.d/10-default-env.conf", "root", 0644, conf),
			},
		},
	}

	rawExt, err := ignition.ConvertToRawExtension(ignConfig)
	if err != nil {
		return nil, err
	}

	return &mcfgv1.MachineConfig{
		TypeMeta: metav1.TypeMeta{
			APIVersion: "machineconfiguration.openshift.io/v1",
			Kind:       "MachineConfig",
		},
		ObjectMeta: metav1.ObjectMeta{
			Name: fmt.Sprintf("99-%s-crio-default-env", role),
			Labels: map[string]string{
				"machineconfiguration.openshift.io/role": role,
			},
		},
		Spec: mcfgv1.MachineConfigSpec{
			Config: rawExt,
		},
	}, nil
}

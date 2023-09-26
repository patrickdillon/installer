package aws

import (
	"encoding/base64"
	"fmt"
	"os"
	"os/exec"

	"github.com/davecgh/go-spew/spew"
	"github.com/pkg/errors"
	"sigs.k8s.io/yaml"
)

func Run(kc []byte) error {
	spew.Dump("===making kubeconfig===")
	spew.Dump(kc)

	spew.Dump(string(kc))
	data, err := yaml.Marshal(kc)
	if err != nil {
		spew.Dump("FAILED MARSHAL===")
		return errors.Wrap(err, "failed to Marshal kubeconfig")
	}
	dst := make([]byte, base64.StdEncoding.DecodedLen(len(data)))
	n, err := base64.StdEncoding.Decode(dst, data)
	if err != nil {
		spew.Dump("DECODING===")
		spew.Dump(err)
		return err
	}
	dst = dst[:n]
	spew.Dump(dst)
	tmpfile, err := os.CreateTemp("", "installer-kubeconfig")
	if err != nil {
		spew.Dump("TEMP!?!===")
		return err
	}

	spew.Dump("done making kubeconfig===")
	//defer os.Remove(tmpfile.Name()) // clean up

	if _, err := tmpfile.Write(dst); err != nil {
		return err
	}
	if err := tmpfile.Close(); err != nil {
		return err
	}
	spew.Dump(tmpfile.Name())

	awsManagerPath := os.Getenv("OPENSHIFT_INSTALL_CAPI_AWS")

	kcArg := fmt.Sprintf("--kubeconfig=%s", tmpfile.Name())
	spew.Dump(kcArg)
	command := exec.Command(awsManagerPath, kcArg, "-v=10")
	command.Stdout = os.Stdout
	command.Stderr = os.Stderr
	err = command.Run()
	if err != nil {
		spew.Dump("ERROR")
		spew.Dump(err)
		return err
	}
	return nil
}

package gather

import (
	"archive/tar"
	"compress/gzip"
	"io"
	"os"

	"github.com/pkg/errors"
	"github.com/sirupsen/logrus"

	"github.com/openshift/installer/pkg/asset/cluster"
	"github.com/openshift/installer/pkg/gather/providers"
)

// New returns a Gather based on `metadata.json` in `rootDir`.
func New(logger logrus.FieldLogger, serialLogBundle string, bootstrap string, masters []string, rootDir string) (providers.Gather, error) {
	metadata, err := cluster.LoadMetadata(rootDir)
	if err != nil {
		return nil, err
	}

	platform := metadata.Platform()
	if platform == "" {
		return nil, errors.New("no platform configured in metadata")
	}

	creator, ok := providers.Registry[platform]
	if !ok {
		return nil, errors.Errorf("no gather methods registered for %q", platform)
	}
	return creator(logger, serialLogBundle, bootstrap, masters, metadata)
}

// CreateArchive creates a gzipped tar file.
func CreateArchive(files []string, archiveName string) error {
	file, err := os.Create(archiveName)
	if err != nil {
		return err
	}
	defer file.Close()

	gzipWriter := gzip.NewWriter(file)
	defer gzipWriter.Close()

	tarWriter := tar.NewWriter(gzipWriter)
	defer tarWriter.Close()

	for _, filename := range files {
		err := addToArchive(tarWriter, filename)
		if err != nil {
			return err
		}
	}

	return nil
}

func addToArchive(tarWriter *tar.Writer, filename string) error {
	file, err := os.Open(filename)
	if err != nil {
		return err
	}
	defer file.Close()

	st, err := file.Stat()
	if err != nil {
		return err
	}

	header, err := tar.FileInfoHeader(st, st.Name())
	if err != nil {
		return err
	}

	header.Name = filename
	err = tarWriter.WriteHeader(header)
	if err != nil {
		return err
	}

	_, err = io.Copy(tarWriter, file)
	if err != nil {
		return err
	}

	return nil
}

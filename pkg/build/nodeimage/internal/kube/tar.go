package kube

import (
	"archive/tar"
	"compress/gzip"
	"fmt"
	"io"
	"os"
	"path/filepath"

	"sigs.k8s.io/kind/pkg/log"
)

// extractTarball takes a gzipped-tarball and extracts the contents into a specified directory
func extractTarball(tarPath, destDirectory string, logger log.Logger) (err error) {
	// Open the tar file
	f, err := os.Open(tarPath)
	if err != nil {
		return fmt.Errorf("opening tarball: %w", err)
	}
	defer f.Close()

	gzipReader, err := gzip.NewReader(f)
	if err != nil {
		return err
	}
	tr := tar.NewReader(gzipReader)

	numFiles := 0
	for {
		hdr, err := tr.Next()
		if err == io.EOF {
			break
		}
		if err != nil {
			return fmt.Errorf("reading tarfile %s: %w", tarPath, err)
		}

		if hdr.FileInfo().IsDir() {
			continue
		}

		if err := os.MkdirAll(
			filepath.Join(destDirectory, filepath.Dir(hdr.Name)), os.FileMode(0o755),
		); err != nil {
			return fmt.Errorf("creating image directory structure: %w", err)
		}

		f, err := os.Create(filepath.Join(destDirectory, hdr.Name))
		if err != nil {
			return fmt.Errorf("creating image layer file: %w", err)
		}

		if _, err := io.CopyN(f, tr, hdr.Size); err != nil {
			f.Close()
			if err == io.EOF {
				break
			}

			return fmt.Errorf("extracting image data: %w", err)
		}
		f.Close()

		numFiles++
	}

	logger.V(2).Infof("Successfully extracted %d files from image tarball %s", numFiles, tarPath)
	return err
}

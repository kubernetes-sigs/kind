/*
Copyright The Kubernetes Authors.

Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

    http://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
*/

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

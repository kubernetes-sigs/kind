/*
Copyright 2018 The Kubernetes Authors.

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

package logs

import (
	"archive/tar"
	"fmt"
	"io"
	"os"
	"path"
	"path/filepath"

	"al.essio.dev/pkg/shellescape"

	"sigs.k8s.io/kind/pkg/cluster/nodes"
	"sigs.k8s.io/kind/pkg/errors"
	"sigs.k8s.io/kind/pkg/exec"
	"sigs.k8s.io/kind/pkg/log"
)

// DumpDir dumps the dir nodeDir on the node to the dir hostDir on the host
func DumpDir(logger log.Logger, node nodes.Node, nodeDir, hostDir string) (err error) {
	cmd := node.Command(
		"sh", "-c",
		// Tar will exit 1 if a file changed during the archival.
		// We don't care about this, so we're invoking it in a shell
		// And masking out 1 as a return value.
		// Fatal errors will return exit code 2.
		// http://man7.org/linux/man-pages/man1/tar.1.html#RETURN_VALUE
		fmt.Sprintf(
			`tar --hard-dereference -C %s -chf - . || (r=$?; [ $r -eq 1 ] || exit $r)`,
			shellescape.Quote(path.Clean(nodeDir)+"/"),
		),
	)

	return exec.RunWithStdoutReader(cmd, func(outReader io.Reader) error {
		if err := untar(logger, outReader, hostDir); err != nil {
			return errors.Wrapf(err, "Untarring %q: %v", nodeDir, err)
		}
		return nil
	})
}

// untar reads the tar file from r and writes it into dir.
func untar(logger log.Logger, r io.Reader, dir string) (err error) {
	tr := tar.NewReader(r)
	// Ensure target dir is absolute and cleaned
	dirAbs, err := filepath.Abs(dir)
	if err != nil {
		return errors.Wrapf(err, "could not get absolute path for extraction dir: %v", err)
	}
	for {
		f, err := tr.Next()

		switch {
		case err == io.EOF:
			// drain the reader, which may have trailing null bytes
			// we don't want to leave the writer hanging
			_, err := io.Copy(io.Discard, r)
			return err
		case err != nil:
			return errors.Wrapf(err, "tar reading error: %v", err)
		case f == nil:
			continue
		}

		rel := filepath.FromSlash(f.Name)
		// Compute absolute extraction path
		abs := filepath.Join(dirAbs, rel)
		absClean, err := filepath.Abs(abs)
		if err != nil {
			return errors.Wrapf(err, "could not get absolute path for extraction file: %v", err)
		}
		// Path traversal check: absClean must be within dirAbs
		// Ensure dirAbs ends with a separator to prevent prefix matching issues
		dirAbsWithSep := dirAbs
		if !filepath.HasSuffix(dirAbs, string(os.PathSeparator)) {
			dirAbsWithSep = dirAbs + string(os.PathSeparator)
		}
		if absClean != dirAbs && !strings.HasPrefix(absClean, dirAbsWithSep) {
			logger.Warnf("tar entry %q contains path traversal, skipping", f.Name)
			continue
		}

		switch f.Typeflag {
		case tar.TypeReg:
			wf, err := os.OpenFile(absClean, os.O_CREATE|os.O_RDWR, os.FileMode(f.Mode))
			if err != nil {
				return err
			}
			n, err := io.Copy(wf, tr)
			if closeErr := wf.Close(); closeErr != nil && err == nil {
				err = closeErr
			}
			if err != nil {
				return errors.Errorf("error writing to %s: %v", absClean, err)
			}
			if n != f.Size {
				return errors.Errorf("only wrote %d bytes to %s; expected %d", n, absClean, f.Size)
			}
		case tar.TypeDir:
			if _, err := os.Stat(absClean); err != nil {
				if err := os.MkdirAll(absClean, 0755); err != nil {
					return err
				}
			}
		default:
			logger.Warnf("tar file entry %s contained unsupported file type %v", f.Name, f.Typeflag)
		}
	}
}

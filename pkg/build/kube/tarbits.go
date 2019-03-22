/*
Copyright 2019 The Kubernetes Authors.

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
	"bytes"
	"compress/gzip"
	"fmt"
	"io"
	"io/ioutil"
	"net/http"
	"net/url"
	"os"
	"path"
	"strings"

	"github.com/pkg/errors"
	log "github.com/sirupsen/logrus"
	apimv "k8s.io/apimachinery/pkg/util/version"

	"sigs.k8s.io/kind/pkg/util"
)

const (
	tarBitsCIBuildURI      = "https://storage.googleapis.com/kubernetes-release-dev"
	tarBitsReleaseBuildURI = "https://storage.googleapis.com/kubernetes-release"
)

// TarBitsExtractorFunc is the signature for functions that are able to extract
// the bits from a source location given by src to a destination location given
// by dst. The map returned should adhere to the same key/value rules as the
// map returned by Bits.Paths.
type TarBitsExtractorFunc func(src, dst string) (map[string]string, error)

// TarBits implements Bits for the tarballs.
type TarBits struct {
	dataPath string
	extract  TarBitsExtractorFunc
	kubeRoot string
	paths    map[string]string
	tempData bool
}

var _ Bits = &TarBits{}

func init() {
	RegisterNamedBits("tar", NewTarBits)
}

// NewTarBits returns a new Bits backed by a version file, the release image
// archives and the kubeadm, kubectl, and kubelet binaries. The given kubeRoot
// may be defined three ways:
//
//   1. A local filesystem path containing the files listed above in a flat
//      structure.
//
//   2. An HTTP address that contains the files listed above in a structure
//      that adheres to the layout of the public GCS buckets kubernetes-release
//      and kubernetes-release-dev. For example, if kubeRoot is set to
//      https://k8s.ci/v1.13 then the following URIs should be valid:
//
//        * https://k8s.ci/v1.13/kubernetes.tar.gz
//        * https://k8s.ci/v1.13/bin/GOOS/GOARCH/kube-apiserver.tar
//        * https://k8s.ci/v1.13/bin/GOOS/GOARCH/kube-controller-manager.tar
//        * https://k8s.ci/v1.13/bin/GOOS/GOARCH/kube-scheduler.tar
//        * https://k8s.ci/v1.13/bin/GOOS/GOARCH/kube-proxy.tar
//        * https://k8s.ci/v1.13/bin/GOOS/GOARCH/kubeadm
//        * https://k8s.ci/v1.13/bin/GOOS/GOARCH/kubectl
//        * https://k8s.ci/v1.13/bin/GOOS/GOARCH/kubelet
//
//   3. A valid semantic version for a released version of Kubernetes or
//      begins with "ci/" or "release/". If the string matches any of these
//      then the value is presumed to be a CI or release build hosted on one
//      of the public GCS buckets.
//
//      This option also supports values ended with ".txt", ex. "ci/latest.txt".
//      In fact, all values beginning with "ci/" or "release/" are first checked
//      to see if there's a matching txt file for that value. For example,
//      if "ci/latest" is provided then before assuming that is a directory,
//      "ci/latest.txt" is queried.
//
// In order to prevent the second two options from conflicting with the first,
// a local file path, using the prefix "file://" explicitly indicates the given
// kubeRoot is a local file path.
func NewTarBits(kubeRoot string) (bits Bits, err error) {
	tarBits := &TarBits{
		kubeRoot: kubeRoot,
	}

	if strings.HasPrefix(kubeRoot, "file://") {
		tarBits.dataPath = kubeRoot
		tarBits.extract = tarBits.extractFromLocalDir
	} else if strings.HasPrefix(kubeRoot, "http:") ||
		strings.HasPrefix(kubeRoot, "https:") {
		tarBits.extract = tarBits.extractFromHTTP
	} else if _, err := apimv.ParseGeneric(kubeRoot); err == nil {
		tarBits.extract = tarBits.extractFromSemVer
	} else if strings.HasPrefix(kubeRoot, "ci/") {
		tarBits.extract = tarBits.extractFromCIBuild
	} else if strings.HasPrefix(kubeRoot, "release/") {
		tarBits.extract = tarBits.extractFromReleaseBuild
	} else {
		tarBits.dataPath = kubeRoot
		tarBits.extract = tarBits.extractFromLocalDir
	}

	// tarBits.dataPath is the root directory to which the bits are extracted.
	// If a local extractor is used then the data path is the same as the
	// given kubeRoot.
	//
	// Otherwise dataPath may be defined explicitly via $KIND_TARBITS.
	//
	// If neither the above is true then dataPath is set to a temp directory
	// and its bits are removed after installation.
	if tarBits.dataPath == "" {
		tarBits.dataPath = os.Getenv("KIND_TARBITS")
		if tarBits.dataPath != "" {
			os.MkdirAll(tarBits.dataPath, 0755)
		} else {
			tempDir, err := ioutil.TempDir("", "")
			if err != nil {
				return nil, errors.Wrap(err, "error creating tarbits temp dir")
			}
			tarBits.tempData = true
			tarBits.dataPath = tempDir
		}
	}

	return tarBits, nil
}

// Build implements Bits.Build
func (b *TarBits) Build() error {
	paths, err := b.extract(b.kubeRoot, b.dataPath)
	if err != nil {
		return errors.Wrapf(
			err, "error extracting bits from %s to %s", b.kubeRoot, b.dataPath)
	}
	b.paths = paths
	return nil
}

// Paths implements Bits.Paths
func (b *TarBits) Paths() map[string]string {
	return b.paths
}

// Install implements Bits.Install
func (b *TarBits) Install(install InstallContext) error {
	kindBinDir := path.Join(install.BasePath(), "bin")

	// symlink the kubernetes binaries into $PATH
	binaries := []string{"kubeadm", "kubelet", "kubectl"}
	for _, binary := range binaries {
		if err := install.Run("ln", "-s",
			path.Join(kindBinDir, binary),
			path.Join("/usr/bin/", binary),
		); err != nil {
			return errors.Wrap(err, "failed to symlink binaries")
		}
	}

	// enable the kubelet service
	kubeletService := path.Join(install.BasePath(), "systemd/kubelet.service")
	if err := install.Run("systemctl", "enable", kubeletService); err != nil {
		return errors.Wrap(err, "failed to enable kubelet service")
	}

	// setup the kubelet dropin
	kubeletDropinSource := path.Join(install.BasePath(), "systemd/10-kubeadm.conf")
	kubeletDropin := "/etc/systemd/system/kubelet.service.d/10-kubeadm.conf"
	if err := install.Run("mkdir", "-p", path.Dir(kubeletDropin)); err != nil {
		return errors.Wrap(err, "failed to configure kubelet service")
	}
	if err := install.Run("cp", kubeletDropinSource, kubeletDropin); err != nil {
		return errors.Wrap(err, "failed to configure kubelet service")
	}

	return nil
}

func (b TarBits) extractFromSemVer(
	src, dst string) (map[string]string, error) {

	return b.extractFromReleaseBuild(
		fmt.Sprintf("%s/release/%s", tarBitsReleaseBuildURI, src), dst)
}

func (b TarBits) extractFromCIBuild(
	src, dst string) (map[string]string, error) {

	uri, err := b.resolveBuildURI(
		fmt.Sprintf("%s/%s", tarBitsCIBuildURI, src), true)
	if err != nil {
		return nil, errors.Wrapf(
			err, "error resolving CI build uri %s", src)
	}
	return b.extractFromHTTP(uri, dst)
}

func (b TarBits) extractFromReleaseBuild(
	src, dst string) (map[string]string, error) {

	uri, err := b.resolveBuildURI(
		fmt.Sprintf("%s/%s", tarBitsReleaseBuildURI, src), false)
	if err != nil {
		return nil, errors.Wrapf(
			err, "error resolving release build uri %s", src)
	}
	return b.extractFromHTTP(uri, dst)
}

func (b TarBits) extractFromHTTP(
	src, dst string) (map[string]string, error) {

	// Read the version from the remote kubernetes tarball.
	var version []byte
	{
		kubeTarballURI := src + "/kubernetes.tar.gz"
		resp, err := http.Get(kubeTarballURI)
		if err != nil {
			return nil, errors.Wrapf(err, "error reading %s", kubeTarballURI)
		}
		if resp.StatusCode != http.StatusOK {
			return nil, errors.Errorf(
				"error reading %s: %s", kubeTarballURI, resp.Status)
		}
		defer resp.Body.Close()

		buf, err := b.readVersionFromKubeTarball(resp.Body)
		if err != nil {
			return nil, errors.Wrapf(
				err, "error reading version from %s", kubeTarballURI)
		}
		version = buf
	}

	// Update the destination path to include the version.
	dst = path.Join(dst, string(version))
	os.MkdirAll(dst, 0755)

	// Write the version file.
	versionPath := path.Join(dst, "version")
	if _, err := os.Stat(versionPath); err != nil {
		if os.IsNotExist(err) {
			if err := ioutil.WriteFile(versionPath, version, 0644); err != nil {
				return nil, errors.Wrapf(
					err, "error writing version to %s", versionPath)
			}
		} else {
			return nil, errors.Wrap(err, versionPath)
		}
	} else {
		log.WithField("dst", versionPath).Debug("already exists")
	}

	// Build the URIs to the image archives and binaries.
	osArkPathSlug := b.getOSAndArchPathSlug()
	dstBinDirPath := path.Join(dst, "bin", osArkPathSlug)
	srcBinDirPath := fmt.Sprintf("%s/bin/%s", src, osArkPathSlug)
	log.WithFields(log.Fields{
		"src":           src,
		"dst":           dst,
		"srcBinDirPath": srcBinDirPath,
		"dstBinDirPath": dstBinDirPath,
	}).Debug("downloading bits")
	downloadables := map[string]string{
		fmt.Sprintf("%s/kube-apiserver.tar", srcBinDirPath):          path.Join(dstBinDirPath, "kube-apiserver.tar"),
		fmt.Sprintf("%s/kube-controller-manager.tar", srcBinDirPath): path.Join(dstBinDirPath, "kube-controller-manager.tar"),
		fmt.Sprintf("%s/kube-scheduler.tar", srcBinDirPath):          path.Join(dstBinDirPath, "kube-scheduler.tar"),
		fmt.Sprintf("%s/kube-proxy.tar", srcBinDirPath):              path.Join(dstBinDirPath, "kube-proxy.tar"),
		fmt.Sprintf("%s/kubeadm", srcBinDirPath):                     path.Join(dstBinDirPath, "kubeadm"),
		fmt.Sprintf("%s/kubectl", srcBinDirPath):                     path.Join(dstBinDirPath, "kubectl"),
		fmt.Sprintf("%s/kubelet", srcBinDirPath):                     path.Join(dstBinDirPath, "kubelet"),
	}

	// Download the files.
	os.MkdirAll(dstBinDirPath, 0755)
	for uri, localFilePath := range downloadables {
		if err := b.copyFromURI(uri, localFilePath); err != nil {
			return nil, errors.Wrapf(
				err, "failed to copy %s to %s", uri, localFilePath)
		}
	}

	return b.extractFromLocalDir(src, dst)
}

func (b TarBits) extractFromLocalDir(
	src, dst string) (map[string]string, error) {

	// Ensure the version file exists. If it doesn't, attempt to read the
	// version from the kubernetes tarball.
	versionPath := path.Join(dst, "version")
	if _, err := os.Stat(versionPath); err != nil {
		if os.IsNotExist(err) {
			kubeTarballPath := path.Join(dst, "kubernetes.tar.gz")
			if _, err := os.Stat(kubeTarballPath); err != nil {
				if os.IsNotExist(err) {
					return nil, errors.Errorf(
						"required: %s or %s", versionPath, kubeTarballPath)
				}
				return nil, errors.Wrap(err, kubeTarballPath)
			}

			kubeTarballFile, err := os.Open(kubeTarballPath)
			if err != nil {
				return nil, errors.Wrap(err, kubeTarballPath)
			}
			defer kubeTarballFile.Close()

			ver, err := b.readVersionFromKubeTarball(kubeTarballFile)
			if err != nil {
				return nil, errors.Wrapf(
					err, "error reading version from %s", kubeTarballPath)
			}

			if err := ioutil.WriteFile(versionPath, ver, 0644); err != nil {
				return nil, errors.Wrapf(
					err, "error writing version to %s", versionPath)
			}
		} else {
			return nil, errors.Wrap(err, versionPath)
		}
	} else {
		log.WithField("dst", versionPath).Debug("already exists")
	}

	// Ensure the kubelet.service file exists.
	kubeletSvcPath := path.Join(dst, "kubelet.service")
	if _, err := os.Stat(kubeletSvcPath); err != nil {
		if os.IsNotExist(err) {
			if err := ioutil.WriteFile(
				kubeletSvcPath, getKubeletServiceBytes(), 0644); err != nil {
				return nil, errors.Wrapf(err, "error writing %s", kubeletSvcPath)
			}
		} else {
			return nil, errors.Wrap(err, kubeletSvcPath)
		}
	} else {
		log.WithField("dst", kubeletSvcPath).Debug("already exists")
	}

	// Ensure the 10-kubeadm.conf file exists.
	kubeadmConfPath := path.Join(dst, "10-kubeadm.conf")
	if _, err := os.Stat(kubeadmConfPath); err != nil {
		if os.IsNotExist(err) {
			if err := ioutil.WriteFile(
				kubeadmConfPath, getTenKubeadmConfBytes(), 0644); err != nil {
				return nil, errors.Wrapf(err, "error writing %s", kubeadmConfPath)
			}
		} else {
			return nil, errors.Wrap(err, kubeadmConfPath)
		}
	} else {
		log.WithField("dst", kubeadmConfPath).Debug("already exists")
	}

	binDirPath := path.Join(dst, "bin", b.getOSAndArchPathSlug())
	paths := map[string]string{
		// version file
		versionPath: "version",
		// kubelet service
		kubeletSvcPath: "systemd/kubelet.service",
		// kubeadm config
		kubeadmConfPath: "systemd/10-kubeadm.conf",
		// docker images
		path.Join(binDirPath, "kube-apiserver.tar"):          "images/kube-apiserver.tar",
		path.Join(binDirPath, "kube-controller-manager.tar"): "images/kube-controller-manager.tar",
		path.Join(binDirPath, "kube-scheduler.tar"):          "images/kube-scheduler.tar",
		path.Join(binDirPath, "kube-proxy.tar"):              "images/kube-proxy.tar",
		// binaries
		path.Join(binDirPath, "kubeadm"): "bin/kubeadm",
		path.Join(binDirPath, "kubectl"): "bin/kubectl",
		path.Join(binDirPath, "kubelet"): "bin/kubelet",
	}

	for k, v := range paths {
		if _, err := os.Stat(k); err != nil {
			return nil, errors.Wrapf(err, "cannot access %s at %s", v, k)
		}
		if strings.HasPrefix(v, "bin/") {
			os.Chmod(k, 0755)
		}
	}

	return paths, nil
}

func (b TarBits) httpHeadOK(uri string) (bool, error) {
	resp, err := http.Head(uri)
	if err != nil {
		return false, errors.Wrapf(err, "HTTP HEAD %s failed", uri)
	}
	if resp.StatusCode != http.StatusOK {
		return false, errors.Errorf("HTTP HEAD %s failed: %s", uri, resp.Status)
	}
	return true, nil
}

func (b TarBits) httpGet(uri string) (int64, io.ReadCloser, error) {
	resp, err := http.Get(uri)
	if err != nil {
		return 0, nil, errors.Wrapf(err, "HTTP GET %s failed", uri)
	}
	if resp.StatusCode != http.StatusOK {
		return 0, nil, errors.Errorf("HTTP GET %s failed: %s", uri, resp.Status)
	}
	return resp.ContentLength, resp.Body, nil
}

func (b TarBits) resolveBuildURI(uri string, ciBuild bool) (string, error) {
	// If the URI doesn't end with ".txt" then see if the URI is already valid.
	if !strings.HasSuffix(uri, ".txt") {

		// If there is a kubernetes tarball available at the root of the URI then it
		// is already a valid URI.
		if ok, _ := b.httpHeadOK(uri + "/kubernetes.tar.gz"); ok {
			return uri, nil
		}

		// The URI wasn't valid, so add ".txt" to the end and let's see if the
		// URI points to a valid build.
		uri = uri + ".txt"
	}

	// Do an HTTP GET and read the version from the txt file.
	_, r, err := b.httpGet(uri)
	if err != nil {
		return "", errors.Wrapf(err, "invalid URI: %s", uri)
	}
	defer r.Close()
	buf, err := ioutil.ReadAll(r)
	if err != nil {
		return "", errors.Wrapf(err, "error reading version from %s", uri)
	}
	version := url.PathEscape(string(bytes.TrimSpace(buf)))

	// Format the URI based on whether or not it's a CI or release build.
	if ciBuild {
		return fmt.Sprintf("%s/ci/%s", tarBitsCIBuildURI, version), nil
	}
	return fmt.Sprintf("%s/release/%s", tarBitsReleaseBuildURI, version), nil
}

// readVersionFromKubeTarball reads the version file from the
// kubernetes.tar.gz file at the given uri.
func (b TarBits) readVersionFromKubeTarball(r io.Reader) ([]byte, error) {

	// Create a gzip reader to process the source reader.
	gzipReader, err := gzip.NewReader(r)
	if err != nil {
		return nil, errors.Wrap(err, "error getting gzip reader")
	}

	// Create a tar reader to process the gzip reader.
	tarReader := tar.NewReader(gzipReader)

	// Iterate until able to read the version file.
	for {
		header, err := tarReader.Next()
		if err != nil {
			if err == io.EOF {
				return nil, errors.New(
					"failed to find version in kube tarball")
			}
			return nil, errors.Wrap(
				err, "error iterating kube tarball")
		}
		if header.Name == "kubernetes/version" {
			version, err := ioutil.ReadAll(tarReader)
			if err != nil {
				return nil, errors.Wrap(
					err, "error reading version from kube tarball")
			}
			return bytes.TrimSpace(version), nil
		}
	}
}

func (b TarBits) getOSAndArchPathSlug() string {
	goos := os.Getenv("GOOS")
	if goos == "" {
		goos = util.GetOS()
	}
	goarch := os.Getenv("GOARCH")
	if goarch == "" {
		goarch = util.GetArch()
	}
	return path.Join(goos, goarch)
}

func (b TarBits) copyFromURI(src, dst string) error {
	size, r, err := b.httpGet(src)
	if err != nil {
		return errors.Wrapf(err, "error getting reader for %s", src)
	}
	defer r.Close()

	// If the file already exists and has the same size as the remote
	// content then do not redownload it.
	if f, err := os.Stat(dst); err == nil {
		if size == f.Size() {
			log.WithFields(log.Fields{
				"dst": dst,
				"src": src,
			}).Debug("already exists")
			return nil
		}
	}

	log.WithFields(log.Fields{
		"dst": dst,
		"src": src,
	}).Debug("downloading file")

	w, err := os.Create(dst)
	if err != nil {
		return errors.Wrapf(err, "error creating %s", dst)
	}
	defer w.Close()

	if _, err := io.Copy(w, r); err != nil {
		return errors.Wrapf(err, "error copying %s to %s", src, dst)
	}

	return nil
}

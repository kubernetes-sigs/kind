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

// Package node implements functionality to build the kind node image
package node

import (
	"fmt"
	"io"
	"os"
	"path"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/pkg/errors"
	log "github.com/sirupsen/logrus"

	"k8s.io/apimachinery/pkg/util/sets"
	"k8s.io/apimachinery/pkg/util/version"

	"sigs.k8s.io/kind/pkg/container/docker"
	"sigs.k8s.io/kind/pkg/exec"
	"sigs.k8s.io/kind/pkg/fs"
	"sigs.k8s.io/kind/pkg/internal/build/kube"
	"sigs.k8s.io/kind/pkg/internal/util/env"
	"sigs.k8s.io/kind/pkg/util/concurrent"
)

// DefaultImage is the default name:tag for the built image
const DefaultImage = "kindest/node:latest"

// DefaultBaseImage is the default base image used
const DefaultBaseImage = "kindest/base:v20190819-26e1eb5@sha256:e609eaa7853289ef603db647ae9568b32093b2347f839a2117d98a08bfc7ab17"

// DefaultMode is the default kubernetes build mode for the built image
// see pkg/build/kube.Bits
const DefaultMode = "docker"

// Option is BuildContext configuration option supplied to NewBuildContext
type Option func(*BuildContext)

// WithImage configures a NewBuildContext to tag the built image with `image`
func WithImage(image string) Option {
	return func(b *BuildContext) {
		b.image = image
	}
}

// WithBaseImage configures a NewBuildContext to use `image` as the base image
func WithBaseImage(image string) Option {
	return func(b *BuildContext) {
		b.baseImage = image
	}
}

// WithMode sets the kubernetes build mode for the build context
func WithMode(mode string) Option {
	return func(b *BuildContext) {
		b.mode = mode
	}
}

// WithKuberoot sets the path to the Kubernetes source directory (if empty, the path will be autodetected)
func WithKuberoot(root string) Option {
	return func(b *BuildContext) {
		b.kubeRoot = root
	}
}

// BuildContext is used to build the kind node image, and contains
// build configuration
type BuildContext struct {
	// option fields
	mode      string
	image     string
	baseImage string
	// non-option fields
	arch     string // TODO(bentheelder): this should be an option
	kubeRoot string
	bits     kube.Bits
}

// NewBuildContext creates a new BuildContext with default configuration,
// overridden by the options supplied in the order that they are supplied
func NewBuildContext(options ...Option) (ctx *BuildContext, err error) {
	// default options
	ctx = &BuildContext{
		mode:      DefaultMode,
		image:     DefaultImage,
		baseImage: DefaultBaseImage,
		arch:      env.GetArch(),
	}
	// apply user options
	for _, option := range options {
		option(ctx)
	}
	if ctx.kubeRoot == "" {
		// lookup kuberoot unless mode == "apt",
		// apt should not fail on finding kube root as it does not use it
		kubeRoot := ""
		if ctx.mode != "apt" {
			kubeRoot, err = kube.FindSource()
			if err != nil {
				return nil, errors.Wrap(err, "error finding kuberoot")
			}
		}
		ctx.kubeRoot = kubeRoot
	}
	// initialize bits
	bits, err := kube.NewNamedBits(ctx.mode, ctx.kubeRoot)
	if err != nil {
		return nil, err
	}
	ctx.bits = bits
	return ctx, nil
}

// Build builds the cluster node image, the sourcedir must be set on
// the BuildContext
func (c *BuildContext) Build() (err error) {
	// ensure kubernetes build is up to date first
	log.Infof("Starting to build Kubernetes")
	if err = c.bits.Build(); err != nil {
		log.Errorf("Failed to build Kubernetes: %v", err)
		return errors.Wrap(err, "failed to build kubernetes")
	}
	log.Infof("Finished building Kubernetes")

	// create tempdir to build the image in
	buildDir, err := fs.TempDir("", "kind-node-image")
	if err != nil {
		return err
	}
	defer os.RemoveAll(buildDir)

	log.Infof("Building node image in: %s", buildDir)

	// populate the kubernetes artifacts first
	if err := c.populateBits(buildDir); err != nil {
		return err
	}

	// then the perform the actual docker image build
	return c.buildImage(buildDir)
}

func (c *BuildContext) populateBits(buildDir string) error {
	// always create bits dir
	bitsDir := path.Join(buildDir, "bits")
	if err := os.Mkdir(bitsDir, 0777); err != nil {
		return errors.Wrap(err, "failed to make bits dir")
	}
	// copy all bits from their source path to where we will COPY them into
	// the dockerfile, see images/node/Dockerfile
	bitPaths := c.bits.Paths()
	for src, dest := range bitPaths {
		realDest := path.Join(bitsDir, dest)
		// NOTE: we use copy not copyfile because copy ensures the dest dir
		if err := fs.Copy(src, realDest); err != nil {
			return errors.Wrap(err, "failed to copy build artifact")
		}
	}

	return nil
}

// returns a set of image tags that will be sideloaded
func (c *BuildContext) getBuiltImages() (sets.String, error) {
	images := sets.NewString()
	for _, path := range c.bits.ImagePaths() {
		tags, err := docker.GetArchiveTags(path)
		if err != nil {
			return nil, err
		}
		images.Insert(tags...)
	}
	return images, nil
}

// BuildContainerLabelKey is applied to each build container
const BuildContainerLabelKey = "io.k8s.sigs.kind.build"

// DockerImageArchives is the path within the node image where image archives
// will be stored.
const DockerImageArchives = "/kind/images"

// private kube.InstallContext implementation, local to the image build
type installContext struct {
	basePath    string
	containerID string
}

var _ kube.InstallContext = &installContext{}

func (ic *installContext) BasePath() string {
	return ic.basePath
}

func (ic *installContext) Run(command string, args ...string) error {
	cmd := exec.Command(
		"docker",
		append(
			[]string{"exec", ic.containerID, command},
			args...,
		)...,
	)
	exec.InheritOutput(cmd)
	return cmd.Run()
}

func (ic *installContext) CombinedOutputLines(command string, args ...string) ([]string, error) {
	cmd := exec.Command(
		"docker",
		append(
			[]string{"exec", ic.containerID, command},
			args...,
		)...,
	)
	return exec.CombinedOutputLines(cmd)
}

func (c *BuildContext) buildImage(dir string) error {
	// build the image, tagged as tagImageAs, using the our tempdir as the context
	log.Info("Starting image build ...")
	// create build container
	// NOTE: we are using docker run + docker commit so we can install
	// debians without permanently copying them into the image.
	// if docker gets proper squash support, we can rm them instead
	// This also allows the KubeBit implementations to perform programmatic
	// install in the image
	containerID, err := c.createBuildContainer(dir)
	cmder := docker.ContainerCmder(containerID)

	// ensure we will delete it
	if containerID != "" {
		defer func() {
			exec.Command("docker", "rm", "-f", "-v", containerID).Run()
		}()
	}
	if err != nil {
		log.Errorf("Image build Failed! Failed to create build container: %v", err)
		return err
	}

	log.Info("Building in " + containerID)

	// helper we will use to run "build steps"
	execInBuild := func(command string, args ...string) error {
		return exec.InheritOutput(cmder.Command(command, args...)).Run()
	}

	// make artifacts directory
	if err = execInBuild("mkdir", "/kind/"); err != nil {
		log.Errorf("Image build Failed! Failed to make directory %v", err)
		return err
	}

	// copy artifacts in
	if err = execInBuild("rsync", "-r", "/build/bits/", "/kind/"); err != nil {
		log.Errorf("Image build Failed! Failed to sync bits: %v", err)
		return err
	}

	// install the kube bits
	ic := &installContext{
		basePath:    "/kind/",
		containerID: containerID,
	}
	if err = c.bits.Install(ic); err != nil {
		log.Errorf("Image build Failed! Failed to install Kubernetes: %v", err)
		return err
	}

	// setup kubelet systemd
	// create the kubelet service
	kubeletService := path.Join(ic.BasePath(), "systemd/kubelet.service")
	if err := createFile(cmder, kubeletService, kubeletServiceContents); err != nil {
		return errors.Wrap(err, "failed to create kubelet service file")
	}

	// enable the kubelet service
	if err := cmder.Command("systemctl", "enable", kubeletService).Run(); err != nil {
		return errors.Wrap(err, "failed to enable kubelet service")
	}

	// setup the kubelet dropin
	kubeletDropin := "/etc/systemd/system/kubelet.service.d/10-kubeadm.conf"
	if err := createFile(cmder, kubeletDropin, kubeadm10conf); err != nil {
		return errors.Wrap(err, "failed to configure kubelet service")
	}

	// ensure we don't fail if swap is enabled on the host
	if err = execInBuild("/bin/sh", "-c",
		`echo "KUBELET_EXTRA_ARGS=--fail-swap-on=false" >> /etc/default/kubelet`,
	); err != nil {
		log.Errorf("Image build Failed! Failed to add kubelet extra args: %v", err)
		return err
	}

	// pre-pull images that were not part of the build
	if err = c.prePullImages(dir, containerID); err != nil {
		log.Errorf("Image build Failed! Failed to pull Images: %v", err)
		return err
	}

	// Save the image changes to a new image
	cmd := exec.Command(
		"docker", "commit",
		// we need to put this back after changing it when running the image
		"--change", `ENTRYPOINT [ "/usr/local/bin/entrypoint", "/sbin/init" ]`,
		containerID, c.image,
	)
	exec.InheritOutput(cmd)
	if err = cmd.Run(); err != nil {
		log.Errorf("Image build Failed! Failed to save image: %v", err)
		return err
	}

	log.Info("Image build completed.")
	return nil
}

func createFile(containerCmder exec.Cmder, filePath, contents string) error {
	// ensure the directory first
	// NOTE: the paths inside the container should use the path package
	// and not filepath (!), we want posixy paths in the linux container, NOT
	// whatever path format the host uses. For paths on the host we use filepath
	if err := containerCmder.Command("mkdir", "-p", path.Dir(filePath)).Run(); err != nil {
		return err
	}

	return containerCmder.Command(
		"cp", "/dev/stdin", filePath,
	).SetStdin(
		strings.NewReader(contents),
	).Run()
}

// must be run after kubernetes has been installed on the node
func (c *BuildContext) prePullImages(dir, containerID string) error {
	// first get the images we actually built
	builtImages, err := c.getBuiltImages()
	if err != nil {
		log.Errorf("Image build Failed! Failed to get built images: %v", err)
		return err
	}

	// helpers to run things in the build container
	cmder := docker.ContainerCmder(containerID)
	inheritOutputAndRun := func(cmd exec.Cmd) error {
		exec.InheritOutput(cmd)
		return cmd.Run()
	}

	// get the Kubernetes version we installed on the node
	// we need this to ask kubeadm what images we need
	rawVersion, err := exec.CombinedOutputLines(cmder.Command("cat", kubernetesVersionLocation))
	if err != nil {
		log.Errorf("Image build Failed! Failed to get Kubernetes version: %v", err)
		return err
	}
	if len(rawVersion) != 1 {
		log.Errorf("Image build Failed! Failed to get Kubernetes version: %v", err)
		return errors.New("invalid kubernetes version file")
	}

	// before Kubernetes v1.12.0 kubeadm requires arch specific images, instead
	// later releases use manifest list images
	// at node boot time we retag our images to handle this where necessary,
	// so we virtually re-tag them here.
	ver, err := version.ParseGeneric(rawVersion[0])
	if err != nil {
		return err
	}

	// get image tag fixing function for this version
	fixRepository := repositoryCorrectorForVersion(ver)

	// correct set of built tags using the same logic we will use to rewrite
	// the tags as we load the archives
	fixedImages := sets.NewString()
	for _, image := range builtImages.List() {
		registry, tag, err := docker.SplitImage(image)
		if err != nil {
			return err
		}
		registry = fixRepository(registry)
		fixedImages.Insert(registry + ":" + tag)
	}
	builtImages = fixedImages
	println("built images")
	println(strings.Join(builtImages.List(), ", "))

	// write the default CNI manifest
	// NOTE: the paths inside the container should use the path package
	// and not filepath (!), we want posixy paths in the linux container, NOT
	// whatever path format the host uses. For paths on the host we use filepath
	if err := inheritOutputAndRun(cmder.Command(
		"mkdir", "-p", path.Dir(defaultCNIManifestLocation),
	)); err != nil {
		log.Errorf("Image build Failed! Failed write default CNI Manifest: %v", err)
		return err
	}
	if err := cmder.Command(
		"cp", "/dev/stdin", defaultCNIManifestLocation,
	).SetStdin(
		strings.NewReader(defaultCNIManifest),
	).Run(); err != nil {
		log.Errorf("Image build Failed! Failed write default CNI Manifest: %v", err)
		return err
	}

	// gets the list of images required by kubeadm
	requiredImages, err := exec.CombinedOutputLines(cmder.Command(
		"kubeadm", "config", "images", "list", "--kubernetes-version", rawVersion[0],
	))
	if err != nil {
		return err
	}

	// all builds should isntall the default CNI images currently
	requiredImages = append(requiredImages, defaultCNIImages...)

	// Create "images" subdir.
	imagesDir := path.Join(dir, "bits", "images")
	if err := os.MkdirAll(imagesDir, 0777); err != nil {
		log.Errorf("Image build Failed! Failed create local images dir: %v", err)
		return errors.Wrap(err, "failed to make images dir")
	}

	fns := []func() error{}
	pulledImages := make(chan string, len(requiredImages))
	for i, image := range requiredImages {
		i, image := i, image // https://golang.org/doc/faq#closures_and_goroutines
		fns = append(fns, func() error {
			if !builtImages.Has(image) {
				fmt.Printf("Pulling: %s\n", image)
				err := docker.Pull(image, 2)
				if err != nil {
					log.Warnf("Failed to pull %s with error: %v", image, err)
				}
				// TODO(bentheelder): generate a friendlier name
				pullName := fmt.Sprintf("%d.tar", i)
				pullTo := path.Join(imagesDir, pullName)
				err = docker.Save(image, pullTo)
				if err != nil {
					return err
				}
				pulledImages <- pullTo
			}
			return nil
		})
	}
	if err := concurrent.Coalesce(fns...); err != nil {
		return err
	}
	close(pulledImages)
	pulled := []string{}
	for image := range pulledImages {
		pulled = append(pulled, image)
	}

	// setup image importer
	importer := newContainerdImporter(cmder)
	if err := importer.Prepare(); err != nil {
		log.Errorf("Image build Failed! Failed to prepare containerd to load images %v", err)
		return err
	}

	// TODO: return this error?
	defer func() {
		if err := importer.End(); err != nil {
			log.Errorf("Image build Failed! Failed to tear down containerd after loading images %v", err)
		}
	}()

	// create a plan of image loading
	loadFns := []func() error{}
	for _, image := range pulled {
		image := image // capture loop var
		loadFns = append(loadFns, func() error {
			f, err := os.Open(image)
			if err != nil {
				return err
			}
			defer f.Close()
			return importer.LoadCommand().SetStdout(os.Stdout).SetStderr(os.Stdout).SetStdin(f).Run()
		})
	}
	for _, image := range c.bits.ImagePaths() {
		image := image // capture loop var
		loadFns = append(loadFns, func() error {
			f, err := os.Open(image)
			if err != nil {
				return err
			}
			defer f.Close()
			//return importer.LoadCommand().SetStdout(os.Stdout).SetStderr(os.Stderr).SetStdin(f).Run()
			// we will rewrite / correct the tags as we load the image
			if err := exec.RunWithStdinWriter(importer.LoadCommand().SetStdout(os.Stdout).SetStderr(os.Stdout), func(w io.Writer) error {
				return docker.EditArchiveRepositories(f, w, fixRepository)
			}); err != nil {
				return err
			}
			return nil
		})
	}

	// run all image loading concurrently until one fails or all succeed
	if err := concurrent.UntilError(loadFns); err != nil {
		log.Errorf("Image build Failed! Failed to load images %v", err)
		return err
	}

	return nil
}

func repositoryCorrectorForVersion(kubeVersion *version.Version) func(string) string {
	// TODO(bentheelder): we assume the host arch, but cross compiling should
	// be possible now
	arch := env.GetArch()
	archSuffix := "-" + arch

	// For kubernetes v1.15+ (actually 1.16 alpha versions) we may need to
	// drop the arch suffix from images to get the expected image
	// for < v1.12 we need to do the opposite.
	// We can accomplish this by just handling < 1.12 & >= 1.12 as we won't
	// touch images that match the expectation in either case ...

	if kubeVersion.LessThan(version.MustParseSemantic("v1.12.0")) {
		return func(repository string) string {
			if !strings.HasSuffix(repository, archSuffix) {
				fixed := repository + archSuffix
				fmt.Println("fixed: " + repository + " -> " + fixed)
				repository = fixed
			}
			return repository
		}
	}

	return func(repository string) string {
		if strings.HasSuffix(repository, archSuffix) {
			fixed := strings.TrimSuffix(repository, archSuffix)
			fmt.Println("fixed: " + repository + " -> " + fixed)
			repository = fixed
		}
		return repository
	}
}

func (c *BuildContext) createBuildContainer(buildDir string) (id string, err error) {
	// attempt to explicitly pull the image if it doesn't exist locally
	// we don't care if this errors, we'll still try to run which also pulls
	_, _ = docker.PullIfNotPresent(c.baseImage, 4)
	id = "kind-build-" + uuid.New().String()
	err = docker.Run(
		c.baseImage,
		docker.WithRunArgs(
			"-d", // make the client exit while the container continues to run
			// label the container to make them easier to track
			"--label", fmt.Sprintf("%s=%s", BuildContainerLabelKey, time.Now().Format(time.RFC3339Nano)),
			"-v", fmt.Sprintf("%s:/build", buildDir),
			// the container should hang forever so we can exec in it
			"--entrypoint=sleep",
			"--name="+id,
		),
		docker.WithContainerArgs(
			"infinity", // sleep infinitely to keep the container around
		),
	)
	if err != nil {
		return id, errors.Wrap(err, "failed to create build container")
	}
	return id, nil
}

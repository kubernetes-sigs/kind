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

// Package testcmd implements generic test command logic
package testcmd

import (
	"fmt"
	"os"
	"os/signal"
	"path/filepath"
	"strconv"
	"strings"
	"sync"
	"syscall"
	"time"

	"github.com/spf13/cobra"

	"sigs.k8s.io/kind/pkg/apis/config/v1alpha4"
	"sigs.k8s.io/kind/pkg/build/node"
	"sigs.k8s.io/kind/pkg/cluster"
	"sigs.k8s.io/kind/pkg/cluster/nodeutils"
	"sigs.k8s.io/kind/pkg/cmd"
	"sigs.k8s.io/kind/pkg/cmd/kind/version"
	"sigs.k8s.io/kind/pkg/errors"
	"sigs.k8s.io/kind/pkg/exec"
	"sigs.k8s.io/kind/pkg/fs"
	"sigs.k8s.io/kind/pkg/log"
)

// flagpole holds our flags
type flagpole struct {
	Name      string
	ImageName string
	Artifacts string
	IPFamily  string
}

func (f *flagpole) kubeconfigPath() string {
	return filepath.Join(f.Artifacts, "kind", "kubeconfig")
}

func (f *flagpole) Process() error {
	// Default Artifacts
	if f.Artifacts == "" {
		f.Artifacts = os.Getenv("ARTIFACTS")
		if f.Artifacts == "" {
			tmpdir, err := fs.TempDir("", "kind-test")
			if err != nil {
				return errors.Wrap(err, "failed to create tempdir for artifacts")
			}
			f.Artifacts = tmpdir
		}
	}
	// Validate
	if f.IPFamily != string(v1alpha4.IPv4Family) && f.IPFamily != string(v1alpha4.IPv6Family) {
		return fmt.Errorf(
			"invalid --ipfamily %q; valid are: %q, %q",
			f.IPFamily, v1alpha4.IPv4Family, v1alpha4.IPv6Family,
		)
	}
	return nil
}

// Details represents injected details of a particular test command
type Details struct {
	Name       string
	UsageLong  string
	UsageShort string
	Focus      string
	Skip       string
	Parallel   bool
}

// New returns a new cobra.Command for Kubernetes testing with ginkgo-e2e.sh
func New(logger log.Logger, streams cmd.IOStreams, details Details) *cobra.Command {
	flags := &flagpole{}
	cmd := &cobra.Command{
		Args:  cobra.NoArgs,
		Use:   details.Name,
		Short: details.UsageShort,
		Long:  details.UsageLong,
		RunE: func(cmd *cobra.Command, args []string) error {
			// flag defaulting & validation
			if err := flags.Process(); err != nil {
				return err
			}

			// handle test command overrides
			if focus, set := os.LookupEnv("KIND_TEST_FOCUS_OVERRIDE"); set {
				details.Focus = focus
				logger.Warnf("KIND_TEST_FOCUS_OVERRIDE overrode --ginkgo.focus to be: %s", focus)
			}
			if skip, set := os.LookupEnv("KIND_TEST_SKIP_OVERRIDE"); set {
				details.Skip = skip
				logger.Warnf("KIND_TEST_SKIP_OVERRIDE overrode --ginkgo.skip to be: %s", skip)
			}

			// actually run the command
			return runGinkgo(logger, flags, details)
		},
	}
	cmd.Flags().StringVar(&flags.IPFamily, "ip-family", string(v1alpha4.IPv4Family), "test cluster IP family")
	cmd.Flags().StringVar(&flags.Name, "name", details.Name, "test cluster context name")
	cmd.Flags().StringVar(&flags.Artifacts, "artifacts-dir", "", "the directory in which to output test results")
	cmd.Flags().StringVar(&flags.ImageName, "image", "kindest/node:kubernetes-presubmit", "name:tag of the built kind node image")
	return cmd
}

// TODO: junit output for steps a la kubetest
func runGinkgo(logger log.Logger, flags *flagpole, details Details) (err error) {
	// first debug the version we're using
	logger.V(0).Infof("Testing with %s", version.DisplayVersion())

	// then build
	if err := buildWithGinkgoAndTests(logger, flags); err != nil {
		return err
	}

	// setup shared provider for doing cluster things
	provider := cluster.NewProvider(
		cluster.ProviderWithLogger(logger),
	)

	// Check if the cluster already exists
	if err := checkAlreadyExists(provider, flags.Name); err != nil {
		return err
	}

	// then before we do anything more, arrange for cleanup to happen
	cleanup := func(calledFromSignalHandler bool) {
		err2 := provider.CollectLogs(flags.Name, filepath.Join(flags.Artifacts, "logs"))
		err3 := provider.Delete(flags.Name, flags.kubeconfigPath())
		// don't touch err if we're being called from another goroutine
		// on program early exit, or if err is already non-nil
		if calledFromSignalHandler || err != nil {
			return
		} else if err2 != nil {
			err = err2
		} else {
			err = err3
		}
	}
	var cleanupOnce sync.Once
	signals := make(chan os.Signal, 1)
	signal.Notify(signals, syscall.SIGTERM, os.Interrupt)
	go func() {
		<-signals
		cleanupOnce.Do(func() { cleanup(true) })
		os.Exit(1)
	}()
	defer func() {
		cleanupOnce.Do(func() { cleanup(false) })
	}()

	// test (including cluster bringup)
	return testGinkgo(logger, provider, flags, details)
}

func buildWithGinkgoAndTests(logger log.Logger, flags *flagpole) error {
	// primarily for CI we allow overriding the build mode
	buildType := os.Getenv("KIND_BUILD_TYPE")
	switch buildType {
	case "bazel", "docker":
	default:
		buildType = "docker"
	}

	// possibly enable bazel build caching before building kubernetes
	// TODO: remove this if we move to RBE ..?
	if os.Getenv("BAZEL_REMOTE_CACHE_ENABLED") == "true" {
		_ = exec.Command("create_bazel_cache_rcs.sh").Run()
	}

	// first build the node image
	ctx, err := node.NewBuildContext(
		node.WithMode(buildType),
		node.WithImage(flags.ImageName),
		node.WithLogger(logger),
	)
	if err != nil {
		return errors.Wrap(err, "error creating build context")
	}
	if err := ctx.Build(); err != nil {
		return errors.Wrap(err, "error building node image")
	}

	// build test binaries
	switch buildType {
	case "bazel":
		if err := exec.InheritOutput(
			exec.Command("bazel", "build", "//test/e2e:e2e.test", "//vendor/github.com/onsi/ginkgo/ginkgo"),
		).Run(); err != nil {
			return err
		}
		// TODO: cleanup or remove this
		// we are getting kubectl into the PATH
		p, err := exec.Output(exec.Command("sh", "-c", `echo "$(dirname "$(find "${PWD}/bazel-bin/" -name kubectl -type f)"):${PATH}"`).SetEnv(os.Environ()...))
		if err != nil {
			return err
		}
		os.Setenv("PATH", strings.TrimSuffix(p, "\n"))

	case "docker":
		if err := exec.InheritOutput(
			exec.Command(
				"build/run.sh",
				"make", "all", `WHAT=cmd/kubectl test/e2e/e2e.test vendor/github.com/onsi/ginkgo/ginkgo`,
			).SetEnv(
				"KUBE_VERBOSE=0",
			),
		).Run(); err != nil {
			return err
		}
	default:
		return errors.Errorf("build mode %q is not implemented yet!", buildType)
	}

	// In Kubernetes's CI: attempt to release some memory after building
	if len(os.Getenv("KUBETEST_IN_DOCKER")) > 0 {
		_ = exec.Command("sync").Run()
		_ = exec.Command("/proc/sys/vm/drop_caches").SetStdin(strings.NewReader("1")).Run()
	}

	return nil
}

func checkAlreadyExists(provider *cluster.Provider, name string) error {
	n, err := provider.ListNodes(name)
	if err != nil {
		return err
	}
	if len(n) != 0 {
		return fmt.Errorf("node(s) already exist for a cluster with the name %q", name)
	}
	return nil
}

func testGinkgo(logger log.Logger, provider *cluster.Provider, flags *flagpole, details Details) error {
	// construct the cluster config
	config := &v1alpha4.Cluster{
		Networking: v1alpha4.Networking{
			IPFamily: v1alpha4.ClusterIPFamily(flags.IPFamily),
		},
		Nodes: []v1alpha4.Node{
			{
				Role:  v1alpha4.ControlPlaneRole,
				Image: flags.ImageName,
			},
		},
	}
	// add e2e workers
	numNodesForE2E := 2
	for i := 0; i < numNodesForE2E; i++ {
		config.Nodes = append(config.Nodes, v1alpha4.Node{
			Role:  v1alpha4.WorkerRole,
			Image: flags.ImageName,
		})
	}

	// create the cluster
	logger.V(0).Infof("Creating cluster %q ...\n", flags.Name)
	if err := provider.Create(
		flags.Name,
		cluster.CreateWithV1Alpha4Config(config),
		cluster.CreateWithRetain(true),
		cluster.CreateWithWaitForReady(time.Minute),
		cluster.CreateWithKubeconfigPath(flags.kubeconfigPath()),
		cluster.CreateWithDisplayUsage(false),
		cluster.CreateWithDisplaySalutation(false),
	); err != nil {
		if errs := errors.Errors(err); errs != nil {
			for _, problem := range errs {
				logger.Errorf("%v", problem)
			}
			return errors.New("aborting due to invalid configuration")
		}
		return errors.Wrap(err, "failed to create cluster")
	}

	// In Kubernetes CI we don't have real IPv6 connectivity so we need to
	// employ some work arounds when testing IPv6 clusters
	// We do this all environments for consistency.
	if config.Networking.IPFamily == v1alpha4.IPv6Family {
		if err := enableIPv6Workarounds(provider, flags.Name); err != nil {
			return err
		}
	}

	// run tests
	skip := details.Skip
	if details.Parallel {
		skip = strings.Join([]string{`\[Serial\]`, skip}, "|")
	}
	// pass through host env and override a few specific ones
	env := append([]string{}, os.Environ()...)
	env = append(env,
		// setting this env prevents ginkgo e2e from trying to run provider setup
		"KUBERNETES_CONFORMANCE_TEST=y",
		// setting these is required to make RuntimeClass tests work ... :/
		"KUBE_CONTAINER_RUNTIME=remote",
		"KUBE_CONTAINER_RUNTIME_ENDPOINT=unix:///run/containerd/containerd.sock",
		"KUBE_CONTAINER_RUNTIME_NAME=containerd",
		// ensure the test uses our kubeconfig
		"KUBECONFIG="+flags.kubeconfigPath(),
		"GINKGO_PARALLEL="+boolToYorN(details.Parallel),
	)
	cmd := exec.Command(
		"./hack/ginkgo-e2e.sh",
		// don't use the horrid e2e.test """provider""" concept
		"--provider=skeleton",
		// need to inform tests of worker count
		"--num-nodes", strconv.Itoa(numNodesForE2E),
		// write test results into artifacts
		"--report-dir", flags.Artifacts,
		// we do our own log dump
		"--disable-log-dump=true",
		// plumb ginkgo options
		"--ginkgo.focus", details.Focus,
		"--ginkgo.skip", skip,
	).SetEnv(env...)
	return exec.InheritOutput(cmd).Run()
}

func boolToYorN(y bool) string {
	if y {
		return "y"
	}
	return "n"
}

func enableIPv6Workarounds(provider *cluster.Provider, name string) error {
	// get a control plane node to use run kubectl from
	nodes, err := provider.ListInternalNodes(name)
	if err != nil {
		return errors.Wrap(err, "failed to get nodes for coreDNS IPv6 CI fixes")
	}
	nodes, err = nodeutils.ControlPlaneNodes(nodes)
	if err != nil {
		return errors.Wrap(err, "failed to get control-plane node for coreDNS IPv6 CI fixes")
	}
	if len(nodes) < 1 {
		return errors.New("failed to locate a kind node for coreDNS IPv6 CI fixes")
	}
	node := nodes[0]

	// "fix" coreDNS configmap
	// TODO: refactor this to not be a shell script.
	/*
		IPv6 clusters need some CoreDNS changes in order to work in k8s CI:
		1. k8s CI doesn´t offer IPv6 connectivity, so CoreDNS should be configured
		to work in an offline environment:
		https://github.com/coredns/coredns/issues/2494#issuecomment-457215452

		2. k8s CI adds following domains to resolv.conf search field:
		c.k8s-prow-builds.internal google.internal.
		CoreDNS should handle those domains and answer with NXDOMAIN instead of SERVFAIL
		otherwise pods stops trying to resolve the domain.
	*/
	const hacks = `export KUBECONFIG=/etc/kubernetes/admin.conf; \
kubectl get -oyaml -n=kube-system configmap/coredns | \
sed \
  -e 's/^.*kubernetes cluster\.local/& internal/' \
  -e '/^.*upstream$/d' \
  -e '/^.*fallthrough.*$/d' \
  -e '/^.*forward . \/etc\/resolv.conf$/d' \
  -e '/^.*loop$/d' | \
kubectl apply -f -`
	if err := node.Command("sh", "-c", hacks).Run(); err != nil {
		return errors.Wrap(err, "failed to patch coreDNS configmap for IPv6 CI fixes")
	}

	return nil
}

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

package context

import (
	"fmt"
	"io"
	"io/ioutil"
	"os"
	"path/filepath"

	"github.com/pkg/errors"

	"k8s.io/client-go/tools/clientcmd"
	clientcmdapi "k8s.io/client-go/tools/clientcmd/api"
	"sigs.k8s.io/kind/pkg/cluster/nodes"
	"sigs.k8s.io/kind/pkg/exec"
	"sigs.k8s.io/kind/pkg/internal/cluster/kubeadm"
	"sigs.k8s.io/kind/pkg/internal/cluster/loadbalancer"
	"sigs.k8s.io/kind/pkg/internal/util/env"
)

// KubeConfigPath returns the path to where the Kubeconfig would be placed
// by kind based on the configuration.
func (c *Context) KubeConfigPath() string {
	// configDir matches the standard directory expected by kubectl etc
	configDir := filepath.Join(env.HomeDir(), ".kube")
	// note that the file name however does not, we do not want to overwrite
	// the standard config, though in the future we may (?) merge them
	fileName := fmt.Sprintf("kind-config-%s", c.Name())
	return filepath.Join(configDir, fileName)
}

// ReadKubeConfigFromNode returns the configuration as it is stored on the node
func (c *Context) ReadKubeConfigFromNode() ([]byte, error) {
	allNodes, err := c.ListNodes()
	if err != nil {
		return nil, err
	}
	node, err := nodes.BootstrapControlPlaneNode(allNodes)
	if err != nil {
		return nil, err
	}

	// get the kubeconfig from the bootstrap node
	var bytes []byte
	cmd := node.Command("cat", "/etc/kubernetes/admin.conf")
	err = exec.RunWithStdoutReader(cmd, func(outReader io.Reader) error {
		bytes, err = ioutil.ReadAll(outReader)
		if err != nil {
			return err
		}
		return err
	})
	if err != nil {
		return nil, errors.Wrap(err, "failed to get kubeconfig from node")
	}
	return bytes, nil
}

// ReadKubeConfig returns the configuration as it is stored and
// update the context so it can be used from an external host.
// The kubeconfig file created by kubeadm internally to the node
// must be modified in order to use the random host port reserved
// for the API server and exposed by the node.
func (c *Context) ReadKubeConfig(apiServerAddress string) (*clientcmdapi.Config, error) {
	return c.readKubeConfig(&apiServerAddress)
}

func (c *Context) readKubeConfig(apiServerAddress *string) (*clientcmdapi.Config, error) {
	bytes, err := c.ReadKubeConfigFromNode()
	if err != nil {
		return nil, err
	}
	kubeconfig, err := clientcmd.Load(bytes)
	if err != nil {
		return nil, err
	}
	if apiServerAddress != nil {
		allNodes, err := c.ListNodes()
		if err != nil {
			return nil, err
		}
		hostPort, err := getAPIServerPort(allNodes)
		if err != nil {
			return nil, err
		}
		authInfoName := c.getAuthInfoName()
		// fix the config file, swapping out the server for the forwarded localhost:port
		kubeconfig.Clusters[c.name].Server = fmt.Sprintf("https://%s:%d", *apiServerAddress, hostPort)
		// rename authentication info to include the cluster name in the auth. infos
		kubeconfig.AuthInfos[authInfoName] = kubeconfig.AuthInfos["kubernetes-admin"]
		delete(kubeconfig.AuthInfos, "kubernetes-admin")
		// update the current context
		kubeconfig.Contexts[kubeconfig.CurrentContext].AuthInfo = authInfoName
	}
	return kubeconfig, err
}

// WriteKubeConfig writes the kubeconfig file to be used from an external host.
// If setContext is true then the context is added to the current user configuration file.
func (c *Context) WriteKubeConfig(apiServerAddress string, setContext bool) error {
	kubeconfig, err := c.readKubeConfig(&apiServerAddress)
	if err != nil {
		return err
	}
	if setContext {
		// ReUse the loading rules from go-client to get the current context file
		configLoadingRules := clientcmd.NewDefaultClientConfigLoadingRules()
		// Take the file with first precedence
		configFile := configLoadingRules.GetLoadingPrecedence()[0]

		// load the current configuration or create an empty one
		config, err := clientcmd.LoadFromFile(configFile)
		if err != nil && !os.IsNotExist(err) {
			return err
		}
		if config == nil {
			config = clientcmdapi.NewConfig()
		}

		// Import configuration
		authInfoName := c.getAuthInfoName()
		config.Clusters[c.name] = kubeconfig.Clusters[c.name]
		config.AuthInfos[authInfoName] = kubeconfig.AuthInfos[authInfoName]
		config.Contexts["kubernetes-admin@"+c.name] = kubeconfig.Contexts["kubernetes-admin@"+c.name]
		config.CurrentContext = "kubernetes-admin@" + c.name

		return clientcmd.WriteToFile(*config, configFile)
	}
	return clientcmd.WriteToFile(*kubeconfig, c.KubeConfigPath())
}

func (c *Context) getAuthInfoName() string {
	return "kubernetes-admin-" + c.name
}

// getAPIServerPort returns the port on the host on which the APIServer
// is exposed
func getAPIServerPort(allNodes []nodes.Node) (int32, error) {
	// select the external loadbalancer first
	node, err := nodes.ExternalLoadBalancerNode(allNodes)
	if err != nil {
		return 0, err
	}
	// node will be nil if there is no load balancer
	if node != nil {
		return node.Ports(loadbalancer.ControlPlanePort)
	}

	// fallback to the bootstrap control plane
	node, err = nodes.BootstrapControlPlaneNode(allNodes)
	if err != nil {
		return 0, err
	}

	return node.Ports(kubeadm.APIServerPort)
}

package util

import (
	"encoding/base64"
	"fmt"
	"io/ioutil"
	"os"
	"path"
	"path/filepath"
	"sync/atomic"

	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/client-go/tools/clientcmd/api"
	"k8s.io/client-go/tools/clientcmd/api/latest"
	"k8s.io/client-go/util/homedir"
	"sigs.k8s.io/yaml"
)

// DefaultKubeconfigFile is the absolute default location of the Kubernetes Config File
var DefaultKubeconfigFile = filepath.Join(homedir.HomeDir(), ".kube", "config")

// CertificateAuthorityPath stores the absolute path of Certificate Authority Cert and
// user's Certificate and Key
type CertificateAuthorityPath struct {
	CertificateAuthorityPath, ClientKeyPath, ClientCertPath string
}

// KubeConfig stores the Cluster and User information read from
// Kubernetes configuration file
type KubeConfig struct {
	Clusters []Clusters `json:"clusters"`
	Users    []Users    `json:"users"`
}

// Clusters stores an item of a slice of clusters stored in a
// Kubernetes configuration file
type Clusters struct {
	Name    string `json:"name"`
	Cluster struct {
		CertAuthData string `json:"certificate-authority-data"`
		Server       string `json:"server"`
	}
}

// Users is an item of a slice of users stored in a
// Kubernetes configuration file
type Users struct {
	Name string
	User struct {
		ClientCertData string `json:"client-certificate-data"`
		ClientKeyData  string `json:"client-key-data"`
	}
}

// CertificateAuthorityData is used to return both clusters and users information
type CertificateAuthorityData struct {
	Clusters map[string]Cluster
	Users    map[string]User
}

// Cluster is an entry containing one entry with cluster information
// CertAuthorityData holds a base64 representation of the CA Certificate
type Cluster struct {
	CertAuthorityData string
	Server            string
}

// User is an entry containing the user of the cluster. It contains the base64 representation
// of client certificate and key
type User struct {
	ClientCertificateData string
	ClientKeyData         string
}

// NewKubeConfig provides is used to store the CA information read from
// the kubeconfig file read created
func NewKubeConfig() *KubeConfig {
	return &KubeConfig{}
}

// CertDataPath reads the base64 data, stores PKI files into $HOME/.kube/<cluster_name>
// and returns the absolute path of where these files are
func (k *KubeConfig) CertDataPath(cad *CertificateAuthorityData) (*CertificateAuthorityPath, error) {
	cdp := CertificateAuthorityPath{}

	pkiDir := filepath.Join(os.Getenv("HOME"), ".kube", k.Clusters[0].Name)

	if err := os.RemoveAll(pkiDir); err != nil {
		return nil, err
	}

	if err := os.MkdirAll(pkiDir, 0755); err != nil {
		return nil, err
	}

	for _, value := range k.Clusters {

		decoded, err := base64.StdEncoding.DecodeString(value.Cluster.CertAuthData)
		if err != nil {
			return nil, err
		}

		if err = ioutil.WriteFile(path.Join(pkiDir, "ca.crt"), decoded, 0640); err != nil {
			return nil, err
		}

		cdp.CertificateAuthorityPath = path.Join(pkiDir, "ca.crt")

	}

	for _, value := range k.Users {

		cdp.ClientCertPath = path.Join(pkiDir, "client-cert.crt")
		decoded, err := base64.StdEncoding.DecodeString(value.User.ClientCertData)
		if err != nil {
			return nil, err
		}

		if err = ioutil.WriteFile(cdp.ClientCertPath, decoded, 0640); err != nil {
			return nil, err
		}

		cdp.ClientKeyPath = path.Join(pkiDir, "client.key")
		decoded, err = base64.StdEncoding.DecodeString(value.User.ClientKeyData)
		if err != nil {
			return nil, err
		}

		if err = ioutil.WriteFile(cdp.ClientKeyPath, decoded, 0400); err != nil {
			return nil, err
		}

	}

	return &cdp, nil
}

// CertData reads the base 64 certificate info from kubeconfig and return a
// a CertificateAuthorityData
func (k *KubeConfig) CertData(kubeconfigPath string) (*CertificateAuthorityData, error) {

	kconfigContent, err := ioutil.ReadFile(kubeconfigPath)

	if err != nil {
		return nil, err
	}

	err = yaml.Unmarshal(kconfigContent, &k)
	if err != nil {
		return nil, err
	}

	caData := CertificateAuthorityData{}

	for _, c := range k.Clusters {
		caData.Clusters = map[string]Cluster{
			c.Name: {
				CertAuthorityData: c.Cluster.CertAuthData,
				Server:            c.Cluster.Server,
			},
		}
	}

	for _, u := range k.Users {
		caData.Users = map[string]User{
			u.Name: {
				ClientCertificateData: u.User.ClientCertData,
				ClientKeyData:         u.User.ClientKeyData,
			},
		}
	}
	return &caData, nil
}

// KubeConfigSetup is the kubeconfig setup
type KubeConfigSetup struct {
	// The name of the cluster for this context
	ClusterName string

	// ClustersServerAddress is the address of the kubernetes cluster
	ClusterServerAddress string

	// ClientCertificate is the path to a client cert for TLS
	ClientCertificate string

	// CertificateAuthority is the path to a cert file for the certificate authority
	CertificateAuthority string

	// ClientKey is the path to a client key file for TLS
	ClientKey string

	// Should the current context be kept when setting up this one
	KeepContext bool

	// Should the certificate files be embedeeded instead of referenced by path
	EmbedCerts bool

	// kubeConfigFile is the path where the kube config is stored
	kubeConfigFile atomic.Value
}

// SetKubeConfigFile sets the kubeconfig file
func (k *KubeConfigSetup) SetKubeConfigFile(kubeConfigFile string) {
	k.kubeConfigFile.Store(kubeConfigFile)
}

// GetKubeConfigFile gets the kubeconfig file
func (k *KubeConfigSetup) GetKubeConfigFile() string {
	return k.kubeConfigFile.Load().(string)
}

// PopulateKubeConfig populates an api.Config object
func PopulateKubeConfig(cfg *KubeConfigSetup, kubecfg *api.Config) error {
	var err error
	clusterName := cfg.ClusterName
	cluster := api.NewCluster()
	cluster.Server = cfg.ClusterServerAddress
	// Always embed Certs
	cluster.CertificateAuthorityData, err = ioutil.ReadFile(cfg.CertificateAuthority)
	if err != nil {
		return nil
	}
	kubecfg.Clusters[clusterName] = cluster

	// user
	userName := clusterName
	user := api.NewAuthInfo()
	// Embed Certs and Key
	user.ClientCertificateData, err = ioutil.ReadFile(cfg.ClientCertificate)
	if err != nil {
		return err
	}
	user.ClientKeyData, err = ioutil.ReadFile(cfg.ClientKey)
	if err != nil {
		return err
	}

	kubecfg.AuthInfos[userName] = user

	// context
	contextName := clusterName
	context := api.NewContext()
	context.Cluster = clusterName
	context.AuthInfo = userName
	kubecfg.Contexts[contextName] = context

	// set this cluster to be the currentContext
	kubecfg.CurrentContext = clusterName

	return nil
}

// SetupKubeConfig reads config from disk, adds the cluster settings, and writes it back
func SetupKubeConfig(cfg *KubeConfigSetup) error {

	// read existing config or create new
	config, err := ReadConfigOrNew(cfg.GetKubeConfigFile())
	if err != nil {
		return nil
	}

	if err = PopulateKubeConfig(cfg, config); err != nil {
		return err
	}

	if err = WriteConfig(config, cfg.GetKubeConfigFile()); err != nil {
		return fmt.Errorf("failed writing kubeconfig: %v", err)
	}

	return nil
}

// ReadConfigOrNew reads a configuration file and decode into a configuration object
func ReadConfigOrNew(filename string) (*api.Config, error) {
	content, err := ioutil.ReadFile(filename)
	if os.IsNotExist(err) {
		return api.NewConfig(), nil
	} else if err != nil {
		return nil, fmt.Errorf("error reading file %s: %v", filename, err)
	}

	// decode config, empty if no bytes
	config, err := decode(content)
	if err != nil {
		return nil, fmt.Errorf("could not read config: %v", err)
	}

	if config.AuthInfos == nil {
		config.AuthInfos = map[string]*api.AuthInfo{}
	}
	if config.Clusters == nil {
		config.Clusters = map[string]*api.Cluster{}
	}
	if config.Contexts == nil {
		config.Contexts = map[string]*api.Context{}
	}

	return config, nil
}

// WriteConfig writes the encoded config object into filename
func WriteConfig(config *api.Config, filename string) error {

	data, err := runtime.Encode(latest.Codec, config)
	if err != nil {
		return fmt.Errorf("could not write to '%s': failed to encode config: %v", filename, err)
	}

	dir := filepath.Dir(filename)
	if _, err := os.Stat(dir); err != nil {
		if err = os.MkdirAll(dir, 0755); err != nil {
			return fmt.Errorf("error creating directory '%s': %v", dir, err)
		}
	}

	// Write encoded config
	if err = ioutil.WriteFile(filename, data, 0600); err != nil {
		return fmt.Errorf("error writing file '%s': %v", filename, err)
	}

	return nil
}

func decode(data []byte) (*api.Config, error) {
	if len(data) == 0 {
		return api.NewConfig(), nil
	}

	config, _, err := latest.Codec.Decode(data, nil, nil)
	if err != nil {
		return nil, fmt.Errorf("error decoding config from data: %v", err)
	}

	return config.(*api.Config), nil // incorrect
}

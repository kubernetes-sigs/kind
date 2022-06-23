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

package nodeutils

import (
	"testing"
)

func TestParseSnapshotter(t *testing.T) {
	// a real, known kind node containerd config
	config := `disabled_plugins = []
	imports = ["/etc/containerd/config.toml"]
	oom_score = 0
	plugin_dir = ""
	required_plugins = []
	root = "/var/lib/containerd"
	state = "/run/containerd"
	version = 2
	
	[cgroup]
	  path = ""
	
	[debug]
	  address = ""
	  format = ""
	  gid = 0
	  level = ""
	  uid = 0
	
	[grpc]
	  address = "/run/containerd/containerd.sock"
	  gid = 0
	  max_recv_message_size = 16777216
	  max_send_message_size = 16777216
	  tcp_address = ""
	  tcp_tls_cert = ""
	  tcp_tls_key = ""
	  uid = 0
	
	[metrics]
	  address = ""
	  grpc_histogram = false
	
	[plugins]
	
	  [plugins."io.containerd.gc.v1.scheduler"]
		deletion_threshold = 0
		mutation_threshold = 100
		pause_threshold = 0.02
		schedule_delay = "0s"
		startup_delay = "100ms"
	
	  [plugins."io.containerd.grpc.v1.cri"]
		disable_apparmor = false
		disable_cgroup = false
		disable_hugetlb_controller = true
		disable_proc_mount = false
		disable_tcp_service = true
		enable_selinux = false
		enable_tls_streaming = false
		ignore_image_defined_volumes = false
		max_concurrent_downloads = 3
		max_container_log_line_size = 16384
		netns_mounts_under_state_dir = false
		restrict_oom_score_adj = false
		sandbox_image = "registry.k8s.io/pause:3.7"
		selinux_category_range = 1024
		stats_collect_period = 10
		stream_idle_timeout = "4h0m0s"
		stream_server_address = "127.0.0.1"
		stream_server_port = "0"
		systemd_cgroup = false
		tolerate_missing_hugetlb_controller = true
		unset_seccomp_profile = ""
	
		[plugins."io.containerd.grpc.v1.cri".cni]
		  bin_dir = "/opt/cni/bin"
		  conf_dir = "/etc/cni/net.d"
		  conf_template = ""
		  max_conf_num = 1
	
		[plugins."io.containerd.grpc.v1.cri".containerd]
		  default_runtime_name = "runc"
		  disable_snapshot_annotations = true
		  discard_unpacked_layers = true
		  no_pivot = false
		  snapshotter = "overlayfs"
	
		  [plugins."io.containerd.grpc.v1.cri".containerd.default_runtime]
			base_runtime_spec = ""
			container_annotations = []
			pod_annotations = []
			privileged_without_host_devices = false
			runtime_engine = ""
			runtime_root = ""
			runtime_type = ""
	
			[plugins."io.containerd.grpc.v1.cri".containerd.default_runtime.options]
	
		  [plugins."io.containerd.grpc.v1.cri".containerd.runtimes]
	
			[plugins."io.containerd.grpc.v1.cri".containerd.runtimes.runc]
			  base_runtime_spec = "/etc/containerd/cri-base.json"
			  container_annotations = []
			  pod_annotations = []
			  privileged_without_host_devices = false
			  runtime_engine = ""
			  runtime_root = ""
			  runtime_type = "io.containerd.runc.v2"
	
			  [plugins."io.containerd.grpc.v1.cri".containerd.runtimes.runc.options]
	
			[plugins."io.containerd.grpc.v1.cri".containerd.runtimes.test-handler]
			  base_runtime_spec = ""
			  container_annotations = []
			  pod_annotations = []
			  privileged_without_host_devices = false
			  runtime_engine = ""
			  runtime_root = ""
			  runtime_type = "io.containerd.runc.v2"
	
			  [plugins."io.containerd.grpc.v1.cri".containerd.runtimes.test-handler.options]
	
		  [plugins."io.containerd.grpc.v1.cri".containerd.untrusted_workload_runtime]
			base_runtime_spec = ""
			container_annotations = []
			pod_annotations = []
			privileged_without_host_devices = false
			runtime_engine = ""
			runtime_root = ""
			runtime_type = ""
	
			[plugins."io.containerd.grpc.v1.cri".containerd.untrusted_workload_runtime.options]
	
		[plugins."io.containerd.grpc.v1.cri".image_decryption]
		  key_model = "node"
	
		[plugins."io.containerd.grpc.v1.cri".registry]
		  config_path = ""
	
		  [plugins."io.containerd.grpc.v1.cri".registry.auths]
	
		  [plugins."io.containerd.grpc.v1.cri".registry.configs]
	
		  [plugins."io.containerd.grpc.v1.cri".registry.headers]
	
		  [plugins."io.containerd.grpc.v1.cri".registry.mirrors]
	
		[plugins."io.containerd.grpc.v1.cri".x509_key_pair_streaming]
		  tls_cert_file = ""
		  tls_key_file = ""
	
	  [plugins."io.containerd.internal.v1.opt"]
		path = "/opt/containerd"
	
	  [plugins."io.containerd.internal.v1.restart"]
		interval = "10s"
	
	  [plugins."io.containerd.metadata.v1.bolt"]
		content_sharing_policy = "shared"
	
	  [plugins."io.containerd.monitor.v1.cgroups"]
		no_prometheus = false
	
	  [plugins."io.containerd.runtime.v1.linux"]
		no_shim = false
		runtime = "runc"
		runtime_root = ""
		shim = "containerd-shim"
		shim_debug = false
	
	  [plugins."io.containerd.runtime.v2.task"]
		platforms = ["linux/amd64"]
	
	  [plugins."io.containerd.service.v1.diff-service"]
		default = ["walking"]
	
	  [plugins."io.containerd.snapshotter.v1.aufs"]
		root_path = ""
	
	  [plugins."io.containerd.snapshotter.v1.btrfs"]
		root_path = ""
	
	  [plugins."io.containerd.snapshotter.v1.devmapper"]
		async_remove = false
		base_image_size = ""
		pool_name = ""
		root_path = ""
	
	  [plugins."io.containerd.snapshotter.v1.native"]
		root_path = ""
	
	  [plugins."io.containerd.snapshotter.v1.overlayfs"]
		root_path = ""
	
	  [plugins."io.containerd.snapshotter.v1.zfs"]
		root_path = ""
	
	[proxy_plugins]
	
	  [proxy_plugins.fuse-overlayfs]
		address = "/run/containerd-fuse-overlayfs.sock"
		type = "snapshot"
	
	[stream_processors]
	
	  [stream_processors."io.containerd.ocicrypt.decoder.v1.tar"]
		accepts = ["application/vnd.oci.image.layer.v1.tar+encrypted"]
		args = ["--decryption-keys-path", "/etc/containerd/ocicrypt/keys"]
		env = ["OCICRYPT_KEYPROVIDER_CONFIG=/etc/containerd/ocicrypt/ocicrypt_keyprovider.conf"]
		path = "ctd-decoder"
		returns = "application/vnd.oci.image.layer.v1.tar"
	
	  [stream_processors."io.containerd.ocicrypt.decoder.v1.tar.gzip"]
		accepts = ["application/vnd.oci.image.layer.v1.tar+gzip+encrypted"]
		args = ["--decryption-keys-path", "/etc/containerd/ocicrypt/keys"]
		env = ["OCICRYPT_KEYPROVIDER_CONFIG=/etc/containerd/ocicrypt/ocicrypt_keyprovider.conf"]
		path = "ctd-decoder"
		returns = "application/vnd.oci.image.layer.v1.tar+gzip"
	
	[timeouts]
	  "io.containerd.timeout.shim.cleanup" = "5s"
	  "io.containerd.timeout.shim.load" = "5s"
	  "io.containerd.timeout.shim.shutdown" = "3s"
	  "io.containerd.timeout.task.state" = "2s"
	
	[ttrpc]
	  address = ""
	  gid = 0
	  uid = 0`
	snapshotter, err := parseSnapshotter(config)
	if err != nil {
		t.Fatalf("unexpected error parsing config: %v", err)
	}
	if snapshotter != "overlayfs" {
		t.Fatalf(`unexpected parsed snapshotter: %q, expected "overlayfs"`, snapshotter)
	}

	// sanity check parsing an empty config
	_, err = parseSnapshotter("")
	if err == nil {
		t.Fatal("expected error parsing empty config")
	}

	// sanity check parsing invalid toml
	_, err = parseSnapshotter("aaaa")
	if err == nil {
		t.Fatal("expected error parsing invalid config")
	}
}

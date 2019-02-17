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

package cri

/*
These types are from
https://github.com/kubernetes/kubernetes/blob/063e7ff358fdc8b0916e6f39beedc0d025734cb1/pkg/kubelet/apis/cri/runtime/v1alpha2/api.pb.go#L183
*/

// Mount specifies a host volume to mount into a container.
// This is a copy of the upstream cri Mount type
// see: k8s.io/kubernetes/pkg/kubelet/apis/cri/runtime/v1alpha2
// It additionally serializes the "propogation" field with the string enum
// names on disk as opposed to the int32 values
// In yaml this looks like:
//  container_path: /foo
//  host_path: /bar
//  readonly: true
//  selinux_relabel: false
//  propagation: PROPAGATION_PRIVATE
// Propogation may be one of:
// PROPAGATION_PRIVATE, PROPAGATION_HOST_TO_CONTAINER, PROPAGATION_BIDIRECTIONAL
type Mount struct {
	// Path of the mount within the container.
	ContainerPath string `protobuf:"bytes,1,opt,name=container_path,json=containerPath,proto3" json:"container_path,omitempty"`
	// Path of the mount on the host. If the hostPath doesn't exist, then runtimes
	// should report error. If the hostpath is a symbolic link, runtimes should
	// follow the symlink and mount the real destination to container.
	HostPath string `protobuf:"bytes,2,opt,name=host_path,json=hostPath,proto3" json:"host_path,omitempty"`
	// If set, the mount is read-only.
	Readonly bool `protobuf:"varint,3,opt,name=readonly,proto3" json:"readonly,omitempty"`
	// If set, the mount needs SELinux relabeling.
	SelinuxRelabel bool `protobuf:"varint,4,opt,name=selinux_relabel,json=selinuxRelabel,proto3" json:"selinux_relabel,omitempty"`
	// Requested propagation mode.
	Propagation MountPropagation `protobuf:"varint,5,opt,name=propagation,proto3,enum=runtime.v1alpha2.MountPropagation" json:"propagation,omitempty"`
}

type MountPropagation int32

const (
	// No mount propagation ("private" in Linux terminology).
	MountPropagation_PROPAGATION_PRIVATE MountPropagation = 0
	// Mounts get propagated from the host to the container ("rslave" in Linux).
	MountPropagation_PROPAGATION_HOST_TO_CONTAINER MountPropagation = 1
	// Mounts get propagated from the host to the container and from the
	// container to the host ("rshared" in Linux).
	MountPropagation_PROPAGATION_BIDIRECTIONAL MountPropagation = 2
)

var MountPropagation_name = map[int32]string{
	0: "PROPAGATION_PRIVATE",
	1: "PROPAGATION_HOST_TO_CONTAINER",
	2: "PROPAGATION_BIDIRECTIONAL",
}
var MountPropagation_value = map[string]int32{
	"PROPAGATION_PRIVATE":           0,
	"PROPAGATION_HOST_TO_CONTAINER": 1,
	"PROPAGATION_BIDIRECTIONAL":     2,
}

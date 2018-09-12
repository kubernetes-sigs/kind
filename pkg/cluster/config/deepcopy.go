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

package config

// TODO(bentheelder): consider kubernetes deep-copy gen
// In the meantime the pattern is:
// - handle nil receiver
// - create a new(OutType)
// - *out = *in to copy plain fields
// - copy pointer fields by calling their DeepCopy
// - copy slices / maps by allocating a new one and performing a copy loop

// DeepCopy returns a deep copy
func (in *Config) DeepCopy() *Config {
	if in == nil {
		return nil
	}
	out := new(Config)
	*out = *in
	out.NodeLifecycle = in.NodeLifecycle.DeepCopy()
	return out
}

// DeepCopy returns a deep copy
func (in *NodeLifecycle) DeepCopy() *NodeLifecycle {
	if in == nil {
		return nil
	}
	out := new(NodeLifecycle)
	if in.PreBoot != nil {
		out.PreBoot = make([]LifecycleHook, len(in.PreBoot))
		for i := range in.PreBoot {
			out.PreBoot[i] = *(in.PreBoot[i].DeepCopy())
		}
	}
	if in.PreKubeadm != nil {
		out.PreKubeadm = make([]LifecycleHook, len(in.PreKubeadm))
		for i := range in.PreKubeadm {
			out.PreKubeadm[i] = *(in.PreKubeadm[i].DeepCopy())
		}
	}
	if in.PostKubeadm != nil {
		out.PostKubeadm = make([]LifecycleHook, len(in.PostKubeadm))
		for i := range in.PostKubeadm {
			out.PostKubeadm[i] = *(in.PostKubeadm[i].DeepCopy())
		}
	}
	if in.PostSetup != nil {
		out.PostSetup = make([]LifecycleHook, len(in.PostSetup))
		for i := range in.PostSetup {
			out.PostSetup[i] = *(in.PostSetup[i].DeepCopy())
		}
	}
	return out
}

// DeepCopy returns a deep copy
func (in *LifecycleHook) DeepCopy() *LifecycleHook {
	if in == nil {
		return nil
	}
	out := new(LifecycleHook)
	*out = *in
	if in.Command != nil {
		out.Command = make([]string, len(in.Command))
		for i := range in.Command {
			out.Command[i] = in.Command[i]
		}
	}
	return out
}

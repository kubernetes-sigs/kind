# Architecture Documentation for kind v0.31.0
## Key Changes
- Default Kubernetes version upgraded to 1.35.0
- Deprecation of cgroup v1 support
- Upgraded helper images to Debian 'trixie'

## Implications
- Users must migrate to cgroup v2
- Future releases will adopt kubeadm v1beta4

## Recommendations
- Pin images by digest for stability
- Prepare for future API changes in kubeadm
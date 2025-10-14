#!/usr/bin/env bash
# Copyright 2025 The Kubernetes Authors.
#
# Licensed under the Apache License, Version 2.0 (the "License");
# you may not use this file except in compliance with the License.
# You may obtain a copy of the License at
#
#     http://www.apache.org/licenses/LICENSE-2.0
#
# Unless required by applicable law or agreed to in writing, software
# distributed under the License is distributed on an "AS IS" BASIS,
# WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
# See the License for the specific language governing permissions and
# limitations under the License.

# E2E test script for kind node add/delete operations
# Tests the full lifecycle of adding and removing nodes from a cluster

set -o errexit -o nounset -o pipefail

# Test configuration
readonly CLUSTER_NAME="kind-node-ops-test"
readonly TEST_WORKER_1="test-worker-1"
readonly TEST_WORKER_2="test-worker-2"
readonly TEST_CP_1="test-cp-1"
readonly TEST_CP_2="test-cp-2"
readonly TIMEOUT="120s"

# Colors for output
readonly RED='\033[0;31m'
readonly GREEN='\033[0;32m'
readonly YELLOW='\033[1;33m'
readonly NC='\033[0m' # No Color

# Cleanup logic for cleanup on exit
CLEANED_UP=false
cleanup() {
  if [ "$CLEANED_UP" = "true" ]; then
    return
  fi
  echo -e "${YELLOW}Cleaning up test resources...${NC}"

  # Delete test cluster if it exists
  if kind get clusters | grep -q "^${CLUSTER_NAME}$"; then
    echo "Deleting test cluster: ${CLUSTER_NAME}"
    kind delete cluster --name="${CLUSTER_NAME}" || true
  fi

  CLEANED_UP=true
}

# Setup signal handlers
trap cleanup INT TERM EXIT

# Logging helpers
log_info() {
  echo -e "${GREEN}[INFO]${NC} $1"
}

log_error() {
  echo -e "${RED}[ERROR]${NC} $1" >&2
}

log_warn() {
  echo -e "${YELLOW}[WARN]${NC} $1"
}

# Test helper functions
wait_for_node_ready() {
  local node_name="$1"
  local max_attempts=30
  local attempt=0

  log_info "Waiting for node ${node_name} to become Ready..."

  while [ $attempt -lt $max_attempts ]; do
    if kubectl get nodes "${node_name}" -o jsonpath='{.status.conditions[?(@.type=="Ready")].status}' | grep -q "True"; then
      log_info "Node ${node_name} is Ready"
      return 0
    fi

    attempt=$((attempt + 1))
    echo "Attempt $attempt/$max_attempts: Node ${node_name} not ready yet, waiting..."
    sleep 5
  done

  log_error "Node ${node_name} did not become Ready within timeout"
  kubectl get nodes "${node_name}" -o wide || true
  return 1
}

verify_node_exists() {
  local node_name="$1"

  if kubectl get nodes "${node_name}" >/dev/null 2>&1; then
    log_info "✓ Node ${node_name} exists in cluster"
    return 0
  else
    log_error "✗ Node ${node_name} does not exist in cluster"
    return 1
  fi
}

verify_node_not_exists() {
  local node_name="$1"

  if ! kubectl get nodes "${node_name}" >/dev/null 2>&1; then
    log_info "✓ Node ${node_name} does not exist in cluster (as expected)"
    return 0
  else
    log_error "✗ Node ${node_name} still exists in cluster"
    return 1
  fi
}

verify_node_ready() {
  local node_name="$1"

  local status
  status=$(kubectl get nodes "${node_name}" -o jsonpath='{.status.conditions[?(@.type=="Ready")].status}' 2>/dev/null || echo "NotFound")

  if [ "$status" = "True" ]; then
    log_info "✓ Node ${node_name} is Ready"
    return 0
  else
    log_error "✗ Node ${node_name} is not Ready (status: $status)"
    kubectl get nodes "${node_name}" -o wide || true
    return 1
  fi
}

verify_node_role() {
  local node_name="$1"
  local expected_role="$2"

  local roles
  roles=$(kubectl get nodes "${node_name}" -o jsonpath='{.metadata.labels.node-role\.kubernetes\.io/.*}' 2>/dev/null || echo "")

  if [ "$expected_role" = "worker" ]; then
    # Worker nodes should not have control-plane role
    if [[ "$roles" != *"control-plane"* ]]; then
      log_info "✓ Node ${node_name} has correct worker role"
      return 0
    else
      log_error "✗ Node ${node_name} incorrectly has control-plane role"
      return 1
    fi
  else
    log_warn "Role verification for ${expected_role} not implemented yet"
    return 0
  fi
}

# Helper function to get container configuration
get_container_config() {
  local node_name="$1"
  local config_path="$2"
  docker inspect "$node_name" --format="{{${config_path}}}" 2>/dev/null || echo "ERROR"
}

# Verify container configuration consistency between nodes
verify_container_config_consistency() {
  local reference_node="$1"
  local test_node="$2"
  local config_name="$3"

  log_info "Verifying ${config_name} consistency between ${reference_node} and ${test_node}..."

  local config_path ref_config test_config
  case "$config_name" in
    "restart_policy")
      config_path=".HostConfig.RestartPolicy"
      ;;
    "cgroupns_mode")
      config_path=".HostConfig.CgroupnsMode"
      ;;
    "security_opts")
      config_path=".HostConfig.SecurityOpt"
      ;;
    "tmpfs_mounts")
      config_path=".HostConfig.Tmpfs"
      ;;
    "privileged")
      config_path=".HostConfig.Privileged"
      ;;
    *)
      log_error "Unknown config type: $config_name"
      return 1
      ;;
  esac

  ref_config=$(get_container_config "$reference_node" "$config_path")
  test_config=$(get_container_config "$test_node" "$config_path")

  if [ "$ref_config" = "ERROR" ] || [ "$test_config" = "ERROR" ]; then
    log_error "✗ Failed to get ${config_name} configuration"
    return 1
  fi

  if [ "$ref_config" = "$test_config" ]; then
    log_info "✓ ${config_name} matches: $ref_config"
    return 0
  else
    log_error "✗ ${config_name} mismatch - ${reference_node}: '$ref_config', ${test_node}: '$test_config'"
    return 1
  fi
}

# Test that container configurations are consistent
test_container_config_consistency() {
  local reference_node="$1"
  local test_node="$2"

  log_info "=== Testing container configuration consistency ==="
  log_info "Reference node: ${reference_node}"
  log_info "Test node: ${test_node}"

  # Test key container configuration settings
  verify_container_config_consistency "$reference_node" "$test_node" "restart_policy" || return 1
  verify_container_config_consistency "$reference_node" "$test_node" "cgroupns_mode" || return 1
  verify_container_config_consistency "$reference_node" "$test_node" "security_opts" || return 1
  verify_container_config_consistency "$reference_node" "$test_node" "tmpfs_mounts" || return 1
  verify_container_config_consistency "$reference_node" "$test_node" "privileged" || return 1

  # Verify proxy environment variables are consistent (if they exist)
  local ref_no_proxy test_no_proxy
  ref_no_proxy=$(docker inspect "$reference_node" --format='{{range $index, $value := .Config.Env}}{{if eq (index (split $value "=") 0) "NO_PROXY"}}{{$value}}{{end}}{{end}}' 2>/dev/null || echo "")
  test_no_proxy=$(docker inspect "$test_node" --format='{{range $index, $value := .Config.Env}}{{if eq (index (split $value "=") 0) "NO_PROXY"}}{{$value}}{{end}}{{end}}' 2>/dev/null || echo "")

  if [ "$ref_no_proxy" = "$test_no_proxy" ]; then
    log_info "✓ NO_PROXY environment variable matches"
  else
    log_error "✗ NO_PROXY mismatch - ${reference_node}: '$ref_no_proxy', ${test_node}: '$test_no_proxy'"
    return 1
  fi

  log_info "✓ Container configuration consistency test passed"
  return 0
}

# Check if we're using Docker provider (node ops only supported on Docker)
check_provider_support() {
  local provider="${KIND_EXPERIMENTAL_PROVIDER:-docker}"
  if [ "$provider" != "docker" ]; then
    log_warn "Node operations are only supported with Docker provider"
    log_warn "Current provider: $provider"
    log_warn "Skipping node operations E2E tests"
    exit 0
  fi
  log_info "Using Docker provider, proceeding with node operations tests"
}

# Test functions
test_cluster_creation() {
  log_info "=== Testing cluster creation ==="

  # Create test cluster
  log_info "Creating test cluster: ${CLUSTER_NAME}"
  kind create cluster --name="${CLUSTER_NAME}" --wait="${TIMEOUT}"

  # Verify cluster is ready
  kubectl cluster-info --context="kind-${CLUSTER_NAME}"

  # Show initial cluster state
  log_info "Initial cluster nodes:"
  kubectl get nodes -o wide

  log_info "✓ Cluster creation test passed"
}

test_add_worker_node() {
  log_info "=== Testing worker node addition ==="

  # Add first worker node
  log_info "Adding worker node: ${TEST_WORKER_1}"
  kind add node "${TEST_WORKER_1}" --name="${CLUSTER_NAME}"

  # Verify node was added and is ready
  verify_node_exists "${TEST_WORKER_1}"
  wait_for_node_ready "${TEST_WORKER_1}"
  verify_node_ready "${TEST_WORKER_1}"
  verify_node_role "${TEST_WORKER_1}" "worker"

  # Test container configuration consistency between control plane and worker
  test_container_config_consistency "${CLUSTER_NAME}-control-plane" "${TEST_WORKER_1}"

  # Add second worker node
  log_info "Adding second worker node: ${TEST_WORKER_2}"
  kind add node "${TEST_WORKER_2}" --name="${CLUSTER_NAME}"

  # Verify second node was added and is ready
  verify_node_exists "${TEST_WORKER_2}"
  wait_for_node_ready "${TEST_WORKER_2}"
  verify_node_ready "${TEST_WORKER_2}"
  verify_node_role "${TEST_WORKER_2}" "worker"

  # Test container configuration consistency between workers
  test_container_config_consistency "${TEST_WORKER_1}" "${TEST_WORKER_2}"

  # Show cluster state with new nodes
  log_info "Cluster nodes after adding workers:"
  kubectl get nodes -o wide

  # Verify we have the expected number of nodes
  local node_count
  node_count=$(kubectl get nodes --no-headers | wc -l | tr -d ' ')
  if [ "$node_count" -eq 3 ]; then
    log_info "✓ Cluster has expected number of nodes: $node_count"
  else
    log_error "✗ Cluster has unexpected number of nodes: $node_count (expected: 3)"
    return 1
  fi

  log_info "✓ Worker node addition test passed"
}

test_delete_worker_node() {
  log_info "=== Testing worker node deletion ==="

  # Delete first worker node
  log_info "Deleting worker node: ${TEST_WORKER_1}"
  kind delete node "${TEST_WORKER_1}" --name="${CLUSTER_NAME}"

  # Verify node was removed from cluster
  verify_node_not_exists "${TEST_WORKER_1}"

  # Verify second node is still present and ready
  verify_node_exists "${TEST_WORKER_2}"
  verify_node_ready "${TEST_WORKER_2}"

  # Show cluster state after deletion
  log_info "Cluster nodes after deleting ${TEST_WORKER_1}:"
  kubectl get nodes -o wide

  # Delete second worker node
  log_info "Deleting second worker node: ${TEST_WORKER_2}"
  kind delete node "${TEST_WORKER_2}" --name="${CLUSTER_NAME}"

  # Verify second node was removed from cluster
  verify_node_not_exists "${TEST_WORKER_2}"

  # Verify we're back to just the control plane
  local node_count
  node_count=$(kubectl get nodes --no-headers | wc -l | tr -d ' ')
  if [ "$node_count" -eq 1 ]; then
    log_info "✓ Cluster has expected number of nodes: $node_count"
  else
    log_error "✗ Cluster has unexpected number of nodes: $node_count (expected: 1)"
    return 1
  fi

  # Show final cluster state
  log_info "Final cluster state:"
  kubectl get nodes -o wide

  log_info "✓ Worker node deletion test passed"
}

test_error_cases() {
  log_info "=== Testing error cases ==="

  # Test adding node with duplicate name
  log_info "Testing duplicate node name rejection..."
  if ! kind add node "${TEST_WORKER_1}" --name="${CLUSTER_NAME}"; then
    log_error "Failed to add initial node for duplicate test"
    return 1
  fi
  verify_node_exists "${TEST_WORKER_1}"

  if kind add node "${TEST_WORKER_1}" --name="${CLUSTER_NAME}" 2>/dev/null; then
    log_error "✗ Adding duplicate node name should have failed"
    return 1
  else
    log_info "✓ Duplicate node name correctly rejected"
  fi

  # Test deleting non-existent node
  log_info "Testing non-existent node deletion..."
  if kind delete node "non-existent-node" --name="${CLUSTER_NAME}" 2>/dev/null; then
    log_error "✗ Deleting non-existent node should have failed"
    return 1
  else
    log_info "✓ Non-existent node deletion correctly rejected"
  fi

  # Test operations on non-existent cluster
  log_info "Testing operations on non-existent cluster..."
  if kind add node "test-node" --name="non-existent-cluster" 2>/dev/null; then
    log_error "✗ Adding node to non-existent cluster should have failed"
    return 1
  else
    log_info "✓ Operation on non-existent cluster correctly rejected"
  fi

  # Clean up the test node we created
  kind delete node "${TEST_WORKER_1}" --name="${CLUSTER_NAME}"
  verify_node_not_exists "${TEST_WORKER_1}"

  log_info "✓ Error cases test passed"
}

test_pod_scheduling() {
  log_info "=== Testing pod scheduling on added nodes ==="

  # Add a worker node
  log_info "Adding worker node for scheduling test: ${TEST_WORKER_1}"
  kind add node "${TEST_WORKER_1}" --name="${CLUSTER_NAME}"
  verify_node_exists "${TEST_WORKER_1}"
  wait_for_node_ready "${TEST_WORKER_1}"

  # Create a simple test pod
  log_info "Creating test pod..."
  kubectl create deployment test-deployment --image=nginx:alpine --replicas=2

  # Wait for pods to be scheduled
  log_info "Waiting for pods to be scheduled..."
  kubectl rollout status deployment/test-deployment --timeout="${TIMEOUT}"

  # Verify pods are running and scheduled on different nodes
  log_info "Pod scheduling results:"
  kubectl get pods -o wide -l app=test-deployment

  # Check if at least one pod is on the worker node
  local pods_on_worker
  pods_on_worker=$(kubectl get pods -l app=test-deployment -o jsonpath='{.items[*].spec.nodeName}' | grep -c "${TEST_WORKER_1}" || echo "0")

  if [ "$pods_on_worker" -gt 0 ]; then
    log_info "✓ At least one pod is scheduled on the worker node"
  else
    log_warn "⚠ No pods scheduled on worker node (this may be normal depending on resources)"
  fi

  # Clean up test deployment
  kubectl delete deployment test-deployment

  # Clean up worker node
  kind delete node "${TEST_WORKER_1}" --name="${CLUSTER_NAME}"
  verify_node_not_exists "${TEST_WORKER_1}"

  log_info "✓ Pod scheduling test completed"
}

# Test adding control plane nodes
test_add_control_plane_node() {
  log_info "=== Testing control plane node addition ==="

  log_info "Adding control plane node: ${TEST_CP_1}"
  kind add node "${TEST_CP_1}" --role control-plane --name="${CLUSTER_NAME}"

  verify_node_exists "${TEST_CP_1}"
  wait_for_node_ready "${TEST_CP_1}"
  verify_node_role "${TEST_CP_1}" "control-plane"

  # Test container configuration consistency between original and new control plane
  test_container_config_consistency "${CLUSTER_NAME}-control-plane" "${TEST_CP_1}"

  log_info "Adding second control plane node: ${TEST_CP_2}"
  kind add node "${TEST_CP_2}" --role control-plane --name="${CLUSTER_NAME}"

  verify_node_exists "${TEST_CP_2}"
  wait_for_node_ready "${TEST_CP_2}"
  verify_node_role "${TEST_CP_2}" "control-plane"

  # Test container configuration consistency between added control plane nodes
  test_container_config_consistency "${TEST_CP_1}" "${TEST_CP_2}"

  log_info "Cluster nodes after adding control plane nodes:"
  kubectl get nodes -o wide

  # Verify cluster has expected number of nodes (1 original + 2 new = 3 control plane nodes)
  local control_plane_count
  control_plane_count=$(kubectl get nodes --no-headers -l node-role.kubernetes.io/control-plane= | wc -l)
  if [ "$control_plane_count" -ne 3 ]; then
    log_error "Expected 3 control plane nodes, got ${control_plane_count}"
    return 1
  fi
  log_info "✓ Cluster has expected number of control plane nodes: ${control_plane_count}"

  log_info "✓ Control plane node addition test passed"
}

# Test deleting control plane nodes
test_delete_control_plane_node() {
  log_info "=== Testing control plane node deletion ==="

  log_info "Deleting control plane node: ${TEST_CP_1}"
  kind delete node "${TEST_CP_1}" --name="${CLUSTER_NAME}"

  verify_node_not_exists "${TEST_CP_1}"
  verify_node_exists "${TEST_CP_2}"
  wait_for_node_ready "${TEST_CP_2}"

  log_info "Cluster nodes after deleting ${TEST_CP_1}:"
  kubectl get nodes -o wide

  log_info "Deleting second control plane node: ${TEST_CP_2}"
  kind delete node "${TEST_CP_2}" --name="${CLUSTER_NAME}"

  verify_node_not_exists "${TEST_CP_2}"

  # Verify cluster has expected number of nodes (only original control plane left)
  local control_plane_count
  control_plane_count=$(kubectl get nodes --no-headers -l node-role.kubernetes.io/control-plane= | wc -l)
  if [ "$control_plane_count" -ne 1 ]; then
    log_error "Expected 1 control plane node remaining, got ${control_plane_count}"
    return 1
  fi
  log_info "✓ Cluster has expected number of control plane nodes: ${control_plane_count}"

  log_info "Final cluster state:"
  kubectl get nodes -o wide

  log_info "✓ Control plane node deletion test passed"
}

# Test control plane node error cases
test_control_plane_error_cases() {
  log_info "=== Testing control plane error cases ==="

  log_info "Testing attempt to delete last control plane node..."
  # Try to delete the last control plane node (should fail)
  if kind delete node "${CLUSTER_NAME}-control-plane" --name="${CLUSTER_NAME}" 2>/dev/null; then
    log_error "Deletion of last control plane node should have been rejected"
    return 1
  else
    log_info "✓ Deletion of last control plane node correctly rejected"
  fi

  log_info "✓ Control plane error cases test passed"
}

# Main test execution
main() {
  log_info "Starting kind node operations E2E tests..."
  log_info "Test cluster: ${CLUSTER_NAME}"
  log_info "Test workers: ${TEST_WORKER_1}, ${TEST_WORKER_2}"
  log_info "Test control plane nodes: ${TEST_CP_1}, ${TEST_CP_2}"

  # Check provider support first
  check_provider_support

  # Check prerequisites
  command -v kind >/dev/null 2>&1 || { log_error "kind binary not found in PATH"; exit 1; }
  command -v kubectl >/dev/null 2>&1 || { log_error "kubectl binary not found in PATH"; exit 1; }
  command -v docker >/dev/null 2>&1 || { log_error "docker binary not found in PATH"; exit 1; }

  # Clean up any existing test cluster
  if kind get clusters | grep -q "^${CLUSTER_NAME}$"; then
    log_warn "Test cluster already exists, deleting..."
    kind delete cluster --name="${CLUSTER_NAME}"
  fi

  # Run tests
  test_cluster_creation
  test_add_worker_node
  test_delete_worker_node
  test_error_cases
  test_pod_scheduling
  test_add_control_plane_node
  test_delete_control_plane_node
  test_control_plane_error_cases

  log_info "🎉 All node operations E2E tests passed!"
}

# Run main function
main "$@"

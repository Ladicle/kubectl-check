#!/bin/bash
# Demo script to showcase kubectl-check with all resource types

set -e

echo "================================"
echo "kubectl-check Demo Script"
echo "================================"
echo ""

# Colors
GREEN='\033[0;32m'
BLUE='\033[0;34m'
NC='\033[0m' # No Color

demo_resource() {
    local resource_type=$1
    local resource_name=$2
    local extra_args=$3
    
    echo -e "${BLUE}Checking ${resource_type}: ${resource_name}${NC}"
    ./kubectl-check ${resource_type} ${resource_name} ${extra_args}
    echo ""
    echo "Press Enter to continue..."
    read
}

# Build kubectl-check if not already built
if [ ! -f "./kubectl-check" ]; then
    echo "Building kubectl-check..."
    go build
fi

# Check if optional CRDs are installed
HAS_JOBSET=false
HAS_GATEWAY_API=false
HAS_KUEUE=false

if kubectl get crd jobsets.jobset.x-k8s.io &>/dev/null; then
    echo -e "${GREEN}✓ JobSet CRD detected${NC}"
    HAS_JOBSET=true
else
    echo -e "${BLUE}ℹ JobSet CRD not found, skipping JobSet demo${NC}"
fi

if kubectl get crd gateways.gateway.networking.k8s.io &>/dev/null; then
    echo -e "${GREEN}✓ Gateway API CRDs detected${NC}"
    HAS_GATEWAY_API=true
else
    echo -e "${BLUE}ℹ Gateway API CRDs not found, skipping Gateway API demo${NC}"
fi

if kubectl get crd workloads.kueue.x-k8s.io &>/dev/null; then
    echo -e "${GREEN}✓ Kueue CRD detected${NC}"
    HAS_KUEUE=true
else
    echo -e "${BLUE}ℹ Kueue CRD not found, skipping Kueue demo${NC}"
fi

echo ""

# Apply all samples
echo -e "${GREEN}Applying all sample resources...${NC}"
kubectl apply --force-conflicts --server-side -f samples/deployment.yaml
kubectl apply --force-conflicts --server-side -f samples/replicaset.yaml
kubectl apply --force-conflicts --server-side -f samples/statefulset.yaml
kubectl apply --force-conflicts --server-side -f samples/daemonset.yaml
kubectl apply --force-conflicts --server-side -f samples/job.yaml
kubectl apply --force-conflicts --server-side -f samples/cronjob.yaml
kubectl apply --force-conflicts --server-side -f samples/service.yaml
kubectl apply --force-conflicts --server-side -f samples/pod.yaml
kubectl apply --force-conflicts --server-side -f samples/failing-pod.yaml

# Apply optional resources if CRDs are available
if [ "$HAS_JOBSET" = true ]; then
    kubectl apply --force-conflicts --server-side -f samples/jobset.yaml
fi

if [ "$HAS_GATEWAY_API" = true ]; then
    kubectl apply --force-conflicts --server-side -f samples/gateway.yaml
    kubectl apply --force-conflicts --server-side -f samples/httproute.yaml
    kubectl apply --force-conflicts --server-side -f samples/grpcroute.yaml
fi

if [ "$HAS_KUEUE" = true ]; then
    kubectl apply --force-conflicts --server-side -f samples/kueue-job.yaml
fi

echo ""
echo "Waiting for resources to be ready..."
sleep 10
echo ""

# Demo each resource type
demo_resource "deployment" "sample-deployment"
demo_resource "replicaset" "sample-replicaset"
demo_resource "statefulset" "sample-statefulset"
demo_resource "daemonset" "sample-daemonset"
demo_resource "job" "sample-job"
demo_resource "cronjob" "sample-cronjob"
demo_resource "service" "sample-service"
demo_resource "pod" "sample-pod"

# Demo optional resources
if [ "$HAS_JOBSET" = true ]; then
    demo_resource "jobset" "sample-jobset"
fi

if [ "$HAS_GATEWAY_API" = true ]; then
    demo_resource "gateway" "sample-gateway"
    demo_resource "httproute" "sample-httproute"
    demo_resource "grpcroute" "sample-grpcroute"
fi

if [ "$HAS_KUEUE" = true ]; then
    demo_resource "job" "sample-kueue-job"
fi

echo -e "${BLUE}Checking pod with all logs (not just failed):${NC}"
./kubectl-check pod sample-pod --show-all-logs --tail=10
echo ""
echo "Press Enter to continue..."
read

demo_resource "pod" "failing-pod" ""

echo -e "${GREEN}Demo completed!${NC}"
echo ""
echo "To cleanup:"
echo "  kubectl delete -f samples/"

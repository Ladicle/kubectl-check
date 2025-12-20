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

# Apply all samples
echo -e "${GREEN}Applying all sample resources...${NC}"
kubectl apply -f samples/deployment.yaml
kubectl apply -f samples/statefulset.yaml
kubectl apply -f samples/daemonset.yaml
kubectl apply -f samples/job.yaml
kubectl apply -f samples/cronjob.yaml
kubectl apply -f samples/pod.yaml
kubectl apply -f samples/failing-pod.yaml

echo ""
echo "Waiting for resources to be ready..."
sleep 10
echo ""

# Demo each resource type
demo_resource "deployment" "sample-deployment"
demo_resource "statefulset" "sample-statefulset"
demo_resource "daemonset" "sample-daemonset"
demo_resource "job" "sample-job"
demo_resource "cronjob" "sample-cronjob"
demo_resource "pod" "sample-pod"

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

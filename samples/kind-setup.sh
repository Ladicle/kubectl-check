#!/bin/bash
# Setup script for kubectl-check demo using kind

set -e

echo "================================"
echo "kubectl-check Kind Setup"
echo "================================"
echo ""

# Colors
GREEN='\033[0;32m'
BLUE='\033[0;34m'
RED='\033[0;31m'
NC='\033[0m' # No Color

# Check if kind is installed
if ! command -v kind &> /dev/null; then
    echo -e "${RED}Error: kind is not installed${NC}"
    echo "Please install kind from: https://kind.sigs.k8s.io/docs/user/quick-start/#installation"
    exit 1
fi

# Check if kubectl is installed
if ! command -v kubectl &> /dev/null; then
    echo -e "${RED}Error: kubectl is not installed${NC}"
    echo "Please install kubectl from: https://kubernetes.io/docs/tasks/tools/"
    exit 1
fi

# Check if docker is running
if ! docker info &> /dev/null; then
    echo -e "${RED}Error: Docker is not running${NC}"
    echo "Please start Docker and try again"
    exit 1
fi

echo -e "${BLUE}Creating local storage directory...${NC}"
mkdir -p /tmp/kind-storage

echo -e "${BLUE}Creating kind cluster...${NC}"
if kind get clusters | grep -q kubectl-check-demo; then
    echo -e "${BLUE}Cluster already exists. Deleting...${NC}"
    kind delete cluster --name kubectl-check-demo
fi

kind create cluster --config samples/kind-cluster.yaml

echo ""
echo -e "${GREEN}✓ Kind cluster created successfully${NC}"
echo ""

# Wait for cluster to be ready
echo -e "${BLUE}Waiting for cluster to be ready...${NC}"
kubectl wait --for=condition=Ready nodes --all --timeout=60s

echo ""
echo -e "${GREEN}✓ Cluster is ready${NC}"
echo ""

# Optional: Install JobSet CRD
read -p "Install JobSet CRD? (y/n) " -n 1 -r
echo
if [[ $REPLY =~ ^[Yy]$ ]]; then
    echo -e "${BLUE}Installing JobSet CRD...${NC}"
    kubectl apply --server-side -f https://github.com/kubernetes-sigs/jobset/releases/download/v0.5.2/manifests.yaml
    echo -e "${GREEN}✓ JobSet CRD installed${NC}"
fi

# Optional: Install Gateway API CRDs
read -p "Install Gateway API CRDs? (y/n) " -n 1 -r
echo
if [[ $REPLY =~ ^[Yy]$ ]]; then
    echo -e "${BLUE}Installing Gateway API CRDs...${NC}"
    kubectl apply -f https://github.com/kubernetes-sigs/gateway-api/releases/download/v1.0.0/standard-install.yaml
    echo -e "${GREEN}✓ Gateway API CRDs installed${NC}"
fi

# Optional: Install Kueue CRDs
read -p "Install Kueue CRDs? (y/n) " -n 1 -r
echo
if [[ $REPLY =~ ^[Yy]$ ]]; then
    echo -e "${BLUE}Installing Kueue...${NC}"
    kubectl apply --server-side -f https://github.com/kubernetes-sigs/kueue/releases/download/v0.6.2/manifests.yaml
    echo -e "${GREEN}✓ Kueue installed${NC}"
    
    # Wait for Kueue to be ready
    echo -e "${BLUE}Waiting for Kueue to be ready...${NC}"
    kubectl wait --for=condition=Available deployment/kueue-controller-manager -n kueue-system --timeout=120s
fi

echo ""
echo -e "${GREEN}================================${NC}"
echo -e "${GREEN}Setup completed successfully!${NC}"
echo -e "${GREEN}================================${NC}"
echo ""
echo "Next steps:"
echo "  1. Build kubectl-check: go build"
echo "  2. Run the demo: ./samples/demo.sh"
echo ""
echo "To cleanup:"
echo "  kind delete cluster --name kubectl-check-demo"

# kubectl-check Sample Manifests

This directory contains sample Kubernetes manifests and scripts for testing and demonstrating kubectl-check functionality.

## Quick Start with Kind

### 1. Setup Kind Cluster

```bash
# Run the automated setup script
./samples/kind-setup.sh
```

This script will:
- Create a kind cluster with 3 nodes (1 control-plane + 2 workers)
- Setup local storage directories
- Optionally install CRDs (JobSet, Gateway API, Kueue)

### 2. Build kubectl-check

```bash
go build
```

### 3. Run Demo

```bash
./samples/demo.sh
```

The demo script will apply all sample manifests and walk you through checking each resource type.

## Manual Setup

### Create Kind Cluster

```bash
kind create cluster --config samples/kind-cluster.yaml
```

### Apply Sample Resources

```bash
# Storage resources (PV/PVC)
kubectl apply -f samples/pvc-pv.yaml

# Core workloads
kubectl apply -f samples/deployment.yaml
kubectl apply -f samples/deployment-with-pvc.yaml
kubectl apply -f samples/statefulset.yaml
kubectl apply -f samples/statefulset-with-pvc.yaml
kubectl apply -f samples/daemonset.yaml
kubectl apply -f samples/job.yaml
kubectl apply -f samples/cronjob.yaml

# Network resources
kubectl apply -f samples/service.yaml

# Pods
kubectl apply -f samples/pod.yaml
kubectl apply -f samples/failing-pod.yaml
```

### Check Resources

```bash
# Check storage resources
./kubectl-check pv local-pv-1
./kubectl-check pvc sample-pvc

# Check workloads with PVC
./kubectl-check deployment sample-deployment-with-pvc
./kubectl-check statefulset sample-statefulset-with-pvc

# Check regular workloads
./kubectl-check deployment sample-deployment
./kubectl-check statefulset sample-statefulset
./kubectl-check job sample-job
./kubectl-check pod sample-pod
```

## Available Samples

### Storage Resources
- **pvc-pv.yaml**: PersistentVolumes and PersistentVolumeClaims with local-storage StorageClass
  - `local-pv-1`, `local-pv-2`, `local-pv-3`: PersistentVolumes using hostPath
  - `sample-pvc`: PersistentVolumeClaim for Deployment
  - `data-sample-statefulset-with-pvc-0`: PVC for StatefulSet

### Workloads with Storage
- **deployment-with-pvc.yaml**: Deployment using PVC for nginx content
- **statefulset-with-pvc.yaml**: StatefulSet with VolumeClaimTemplate

### Core Workload Resources

- **deployment.yaml** - Sample Deployment with 3 replicas
- **replicaset.yaml** - ReplicaSet with pod-template-hash label
- **statefulset.yaml** - StatefulSet with 2 replicas and headless service
- **daemonset.yaml** - DaemonSet running on all nodes
- **job.yaml** - Batch Job with completions and parallelism
- **cronjob.yaml** - CronJob running every 5 minutes
- **jobset.yaml** - JobSet with multiple replicated jobs (requires JobSet operator)
- **kueue-job.yaml** - Job with Kueue workload management

### Pod Resources

- **pod.yaml** - Simple Pod with nginx
- **pod-with-gpu.yaml** - Pod with GPU resource requests
- **failing-pod.yaml** - Pod that fails for testing error logs

### Network Resources

- **service.yaml** - ClusterIP and LoadBalancer service examples

### Gateway API Resources

- **gateway.yaml** - Gateway with HTTP and HTTPS listeners (requires Gateway API)
- **httproute.yaml** - HTTPRoute with path-based routing rules (requires Gateway API)
- **grpcroute.yaml** - GRPCRoute with gRPC method routing (requires Gateway API)

## PVC/PV Features

kubectl-check now supports checking PersistentVolumeClaims and PersistentVolumes:

### Direct Resource Checking
```bash
# Check PersistentVolume
./kubectl-check pv local-pv-1
# Shows: Status, Claim, StorageClass, Capacity, AccessModes, ReclaimPolicy

# Check PersistentVolumeClaim
./kubectl-check pvc sample-pvc
# Shows: Status, Volume (if bound), StorageClass, Capacity, AccessModes
# Includes bound PV as child resource
```

### Hierarchical PVC Collection

Parent resources automatically collect and display their PVCs:

- **Deployment**: Shows PVCs referenced in pod template volumes
- **StatefulSet**: Shows PVCs from VolumeClaimTemplates (pattern: `<vct-name>-<sts-name>-<index>`)
- **Job**: Shows PVCs referenced in pod template
- **JobSet**: Shows PVCs from replicatedJobs templates

This avoids duplicate collection - if a Deployment shows its PVC, the ReplicaSet and Pods won't re-fetch the same PVC.

## Cleanup

```bash
# Delete kind cluster
kind delete cluster --name kubectl-check-demo

# Or delete individual resources
kubectl delete -f samples/
```

## Requirements

- **kind**: For creating local Kubernetes cluster
- **kubectl**: For applying resources
- **Docker**: Required by kind
- **Go 1.21+**: For building kubectl-check

## Notes

- The kind cluster uses local hostPath storage (`/tmp/kind-storage` on host)
- PVs use `volumeBindingMode: WaitForFirstConsumer` for proper node affinity
- Some samples require optional CRDs (JobSet, Gateway API, Kueue)
- The demo script detects available CRDs and skips unavailable resources

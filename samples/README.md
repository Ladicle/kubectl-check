# kubectl-check Sample Manifests

This directory contains sample Kubernetes manifests for testing kubectl-check with various resource types.

## Available Samples

### Workload Resources

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

### Service Resources

- **service.yaml** - ClusterIP and LoadBalancer service examples

### Gateway API Resources

- **gateway.yaml** - Gateway with HTTP and HTTPS listeners (requires Gateway API)
- **httproute.yaml** - HTTPRoute with path-based routing rules (requires Gateway API)
- **grpcroute.yaml** - GRPCRoute with gRPC method routing (requires Gateway API)

## Usage

### Apply all samples:
```bash
kubectl apply -f samples/
```

### Apply specific resource:
```bash
kubectl apply -f samples/deployment.yaml
```

### Check resources with kubectl-check:
```bash
# Deployment
./kubectl-check deployment sample-deployment

# ReplicaSet
./kubectl-check replicaset sample-replicaset

# StatefulSet
./kubectl-check statefulset sample-statefulset

# DaemonSet
./kubectl-check daemonset sample-daemonset

# Job
./kubectl-check job sample-job

# CronJob
./kubectl-check cronjob sample-cronjob

# JobSet (if JobSet operator is installed)
./kubectl-check jobset sample-jobset

# Service
./kubectl-check service sample-service

# Gateway (if Gateway API is installed)
./kubectl-check gateway sample-gateway

# HTTPRoute (if Gateway API is installed)
./kubectl-check httproute sample-httproute

# GRPCRoute (if Gateway API is installed)
./kubectl-check grpcroute sample-grpcroute

# Pod
./kubectl-check pod sample-pod

# Pod with GPU
./kubectl-check pod sample-pod-with-gpu

# Failing Pod (to see failed container logs)
./kubectl-check pod failing-pod

# Kueue-managed Job (if Kueue is installed)
./kubectl-check job sample-kueue-job
```

### Show all logs (not just failed containers):
```bash
./kubectl-check pod sample-pod --show-all-logs
```

### Adjust log tail lines:
```bash
./kubectl-check pod sample-pod --show-all-logs --tail=50
```

## Cleanup

Remove all sample resources:
```bash
kubectl delete -f samples/
```

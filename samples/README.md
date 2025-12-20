# kubectl-check Sample Manifests

This directory contains sample Kubernetes manifests for testing kubectl-check with various resource types.

## Available Samples

### Workload Resources

- **deployment.yaml** - Sample Deployment with 3 replicas
- **statefulset.yaml** - StatefulSet with 2 replicas and persistent volumes
- **daemonset.yaml** - DaemonSet running on all nodes
- **job.yaml** - Batch Job with completions and parallelism
- **cronjob.yaml** - CronJob running every 5 minutes

### Pod Resources

- **pod.yaml** - Simple Pod with nginx
- **pod-with-gpu.yaml** - Pod with GPU resource requests
- **failing-pod.yaml** - Pod that fails for testing error logs

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

# StatefulSet
./kubectl-check statefulset sample-statefulset

# DaemonSet
./kubectl-check daemonset sample-daemonset

# Job
./kubectl-check job sample-job

# CronJob
./kubectl-check cronjob sample-cronjob

# Pod
./kubectl-check pod sample-pod

# Pod with GPU
./kubectl-check pod sample-pod-with-gpu

# Failing Pod (to see failed container logs)
./kubectl-check pod failing-pod
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

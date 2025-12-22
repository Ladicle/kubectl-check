# kubectl-check

`kubectl-check` is a kubectl plugin that performs hierarchical checks of Kubernetes resources. It traverses the resource tree downward (from parent resources to pods, containers, events, and logs) and optionally upward (following owner references to parent resources), providing a comprehensive view of resource status and health.

## Installation

Homebrew:

```bash
$ brew install Ladicle/tap/kubectl-check
```

Go:

```bash
$ go install github.com/Ladicle/kubectl-check@latest
```

## Usage

```bash
$ kubectl check -h
Check Kubernetes resource status

Usage:
  kubectl check [flags...] <resource> <name>

Resources:
  - cronjobs, cronjob, cj
  - daemonsets, daemonset, ds
  - deployments, deployment, deploy
  - gateways, gateway, gtw (Gateway API)
  - grpcroutes, grpcroute (Gateway API)
  - httproutes, httproute (Gateway API)
  - jobs, job
  - jobsets, jobset, js
  - services, service, svc
  - pods, pod, po
  - replicasets, replicaset, rs
  - statefulsets, statefulset, sts
  - workloads, workload, wl (Kueue)

Flags:
  -O, --check-owner        Recursively traverse owner references upward
      --show-all-logs      Show logs for all containers (not just failed ones)
      --tail int           Number of log lines to display (default 100)
      --version            Version for check
      --options            Show full options of this command
  -h, --help               Show this message
  -R, --color              Enable color output even if stdout is not a terminal

Available Commands:
  completion   Generate shell completion scripts

Use "kubectl check --options" for full information about global flags.
```

## Shell completion

If you have installed `kubectl-check` via non-Homebrew methods, you need to set up shell completion manually. To do so, download the `kubectl_complete-check` script and place it in your PATH:

```bash
# Download the kubectl_complete-check script from GitHub
curl -LO https://raw.githubusercontent.com/Ladicle/kubectl-check/master/kubectl_complete-check
chmod +x kubectl_complete-check
sudo mv kubectl_complete-check /usr/local/bin/
```

Once installed, you can use completions like:
```bash
# Complete resource types
kubectl check <TAB>
# Output: deployment, pod, statefulset, daemonset, replicaset...

# Complete resource names
kubectl check deploy<TAB>
# Output: backend, frontend, api-server...

# Complete namespaces for the --namespace flag
kubectl check --namespace <TAB>
# Output: default, kube-system, kube-public...
```

For standalone `kubectl-check` command (without kubectl plugin), you can also use Cobra's generated completion:
```bash
# For zsh
source <(kubectl-check completion zsh)

# For bash
source <(kubectl-check completion bash)

# For fish
kubectl-check completion fish | source
```

## Features

### Hierarchical Resource Checking

The plugin checks resources hierarchically, traversing from parent resources down to pods, containers, events, and logs:

```bash
$ kubectl check deploy hello
Deployment/hello                    # Parent resource
  ├─ ReplicaSet/hello-7b6f74959b    # Child resource
  │   └─ Pod/hello-7b6f74959b-6qmh6 # Grandchild resource
  │       └─ Container status
  ├─ Events                         # Events for all resources in hierarchy
  └─ Container Logs                 # Logs from (failed) containers
```

### Owner Reference Traversal

Use `--check-owner` (or `-O`) to traverse owner references upward, showing the complete resource hierarchy:

```bash
$ kubectl check pod my-pod-abc123 --check-owner
JobSet/training-job                 # Top-level owner
  └─ Job/training-job-worker-0      # Intermediate owner
      └─ Pod/my-pod-abc123          # Target resource
          ├─ Container status
          └─ Events and logs
```

### Container Logs

By default, logs from failed containers are displayed. Use flags to customize log output:

```bash
# Show logs from all containers (not just failed ones)
$ kubectl check deploy myapp --show-all-logs

# Limit number of log lines
$ kubectl check pod mypod --tail 50
```

### Kueue Integration

Automatically detects and displays Kueue Workloads as intermediate nodes when pods have the `kueue.x-k8s.io/workload` annotation:

```bash
$ kubectl check job batch-job
Job/batch-job
  └─ Workload/job-batch-job-12345   # Automatically inserted when annotation present
      └─ Pod/batch-job-abc123
```

## Getting Started

```bash
$ kubectl create deployment hello --image=not/found
deployment.apps/hello created

$ kubectl check deploy hello
Deployment/hello
  Namespace:   default
  Pods Status: 0 Ready / 1 Up-to-date / 0 Available (of 1 desired)
  Images:      not/found
  Age:         22m
  ReplicaSet/hello-7b6f74959b
    Pods Status: 0 Ready / 1 Current (of 1 desired)
    Age:         22m
    Pod/hello-7b6f74959b-6qmh6
      Ready:    0/1
      Status:   Pending
      Restarts: 0
      Age:      22m
      IP:       10.244.0.11
      Node:     kind-control-plane

Events:
  Reason               Age      From              Object                                             Message
  ------               ----     ----              ------                                             -------
  ScalingReplicaSet    22m      DeploymentCo...   Deployment/hello                                   Scaled up replica set hello-7b6f74959b from 0 to 1
  SuccessfulCreate     22m      ReplicasetCo...   ReplicaSet/hello-7b6f74959b                        Created pod: hello-7b6f74959b-6qmh6
  Scheduled            22m      DefaultSched...   Pod/hello-7b6f74959b-6qmh6                         Successfully assigned default/hello-7b6f74959b-6qmh6 to kind-control-plane
  Pulling              19m      Kubelet,Kind...   Pod/hello-7b6f74959b-6qmh6/spec.containers{found}  Pulling image "not/found"
  Failed               19m      Kubelet,Kind...   Pod/hello-7b6f74959b-6qmh6/spec.containers{found}  Failed to pull image "not/found": rpc error: code = Unknown desc = failed to pull and unpack image

Failed Container Logs:
Pod/hello-7b6f74959b-6qmh6/found
--------------------------------------------------------------------------------
Error: ImagePullBackOff
```

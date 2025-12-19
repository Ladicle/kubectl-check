# kubectl-check

`kubectl-check` is a kubectl plugin that checks Kubernetes resources. 
Currently it supports deployment, daemonset, statefulset, replicaset, and pod.

## Installation

```bash
$ brew install Ladicle/brew/kubectl-check
```

## Usage

```bash
$ kubectl check -h
Check Kubernetes resource status

Usage:
  kubectl check [flags...] <resource> <name>

Resources:
  - daemonsets, daemonset, ds
  - deployments, deployment, deploy
  - pods, pod, po
  - replicasets, replicaset, rs
  - statefulsets, statefulset, sts

Flags:
  --version    Version for check
  --options    Show full options of this command
  -h, --help   Show this message
  -R, --color  Enable color output even if stdout is not a terminal

Available Commands:
  completion   Generate shell completion scripts

Use "kubectl check --options" for full information about global flags.
```

## Shell completion

This plugin supports shell completion when used through kubectl (requires kubectl v1.26 or higher). To enable shell completion for the plugin, you must copy the `kubectl_complete-check` script somewhere on your `$PATH` and give it executable permissions.

**Note:** If you installed via Homebrew, you still need to manually install the completion script as it is not included in the brew package.

```bash
# Download the kubectl_complete-check script from GitHub
curl -LO https://raw.githubusercontent.com/Ladicle/kubectl-check/master/kubectl_complete-check
chmod +x kubectl_complete-check
sudo mv kubectl_complete-check /usr/local/bin/
```

The `kubectl_complete-check` script provides completions by:
1. Using the builtin `__complete` command provided by [Cobra](https://github.com/spf13/cobra)
2. Leveraging `ValidArgsFunction` to dynamically fetch resource names from your Kubernetes cluster

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
  Reason               Age      From    　　     Object                                             Message
  ------               ----     ----        　　 ------                                             -------
  ScalingReplicaSet    22m      DeploymentCo... Deployment/hello                                   Scaled up replica set hello-7b6f74959b from 0 to 1
  SuccessfulCreate     22m      ReplicasetCo... ReplicaSet/hello-7b6f74959b                        Created pod: hello-7b6f74959b-6qmh6
  Scheduled            22m      DefaultSched... Pod/hello-7b6f74959b-6qmh6                         Successfully assigned default/hello-7b6f74959b-6qmh6 to kind-control-plane
  Pulling              19m      Kubelet,Kind... Pod/hello-7b6f74959b-6qmh6/spec.containers{found}  Pulling image "not/found"
  Failed               19m      Kubelet,Kind... Pod/hello-7b6f74959b-6qmh6/spec.containers{found}  Failed to pull image "not/found": failed to pull and unpack image "docker.io/not/found:latest": failed to resolve reference "docker.io/not/found:latest": pull access denied, repository does not exist or may require authorization: server message: insufficient_scope: authorization failed
  Failed               19m      Kubelet,Kind... Pod/hello-7b6f74959b-6qmh6/spec.containers{found}  Error: ErrImagePull
  BackOff              2m40s    Kubelet,Kind... Pod/hello-7b6f74959b-6qmh6/spec.containers{found}  Back-off pulling image "not/found"
  Failed               2m40s    Kubelet,Kind... Pod/hello-7b6f74959b-6qmh6/spec.containers{found}  Error: ImagePullBackOff
```

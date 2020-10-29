# kubectl-check

`kubectl-check` is a kubectl plugin that checks Kubernetes resources. 
Currently it supports deployment, daemonset and statefulset.

## Installation

```bash
$ brew install Ladicle/brew/kubectl-check
```

## Usage

```bash
$ kubectl check -h
Check Kubernetes resource status

Usage:
  check [flags...] <resource> <name>

Resources:
  - daemonset, ds
  - deployment, deploy, dp
  - statefulset, ststig

Flags:
  --version    Version for check
  --options    Show full options of this command
  -h, --help   Show this message
  -R, --color  Enable color output even if stdout is not a terminal

Use "check --options" for full information about global flags.
Use "check [resource] --help" for more information about each resource.
```

## Getting Started

```bash
$ kubectl create deployment hello --image=not/found
deployment.apps/hello created

$ kubectl check deploy hello
Deployment "default/hello" is not available (0/1):

[ErrImagePull] Pod/hello-7d8df5b78-5zj6x/Container{found}: rpc error: code = Unknown desc = Error response from daemon: pull access denied for not/found, repository does not exist or may require 'docker login': denied: requested access to the resource is denied (restarted x0)

Reason  Age   From              Object                                            Message
------  ----  ----              ------                                            -------
Failed  4s    kubelet, worker2  Pod/hello-7d8df5b78-5zj6x/spec.containers{found}  Failed to pull image "not/found": rpc error: code = Unknown desc = Error response from daemon: pull access denied for not/found, repository does not exist or may require 'docker login': denied: requested access to the resource is denied
Failed  4s    kubelet, worker2  Pod/hello-7d8df5b78-5zj6x/spec.containers{found}  Error: ErrImagePull
Failed  4s    kubelet, worker2  Pod/hello-7d8df5b78-5zj6x/spec.containers{found}  Error: ImagePullBackOff
```

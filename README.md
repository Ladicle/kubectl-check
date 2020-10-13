# kubectl-diagnose

`kubectl-diagnose` is a kubectl plugin that diagnoses Kubernetes resources. 
Currently it supports deployment, daemonset and statefulset.

## Installation

```bash
$ brew install Ladicle/brew/kubectl-diagnose
```

## Usage

```bash
$ kubectl diagnose
Diagnose Kubernetes resource status

Usage:
  diagnose
  diagnose [command]

Available Commands:
  daemonset   Diagnose DaemonSet resource
  deployment  Diagnose Deployment resource
  help        Help about any command
  statefulset Diagnose StatefulSet resource
...
```

## Getting Started

```bash
$ kubectl create deployment hello --image=not/found
deployment.apps/hello created

$ kubectl diagnose deploy hello
Deployment "default/hello" is not available (0/1):

[ErrImagePull] Pod/hello-7d8df5b78-5zj6x/Container{found}: rpc error: code = Unknown desc = Error response from daemon: pull access denied for not/found, repository does not exist or may require 'docker login': denied: requested access to the resource is denied (restarted x0)

Reason  Age   From              Object                                            Message
------  ----  ----              ------                                            -------
Failed  4s    kubelet, worker2  Pod/hello-7d8df5b78-5zj6x/spec.containers{found}  Failed to pull image "not/found": rpc error: code = Unknown desc = Error response from daemon: pull access denied for not/found, repository does not exist or may require 'docker login': denied: requested access to the resource is denied
Failed  4s    kubelet, worker2  Pod/hello-7d8df5b78-5zj6x/spec.containers{found}  Error: ErrImagePull
Failed  4s    kubelet, worker2  Pod/hello-7d8df5b78-5zj6x/spec.containers{found}  Error: ImagePullBackOff
```

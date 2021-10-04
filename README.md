![](https://github.com/supercaracal/kubernetes-controller-template/workflows/Test/badge.svg?branch=master)
![](https://github.com/supercaracal/kubernetes-controller-template/workflows/Release/badge.svg)

Kubernetes Controller Template
===============================================================================

This controller has a feature to create a pod to log a message declared by manifest.
The pod will be deleted automatically by the controller later.

## Running controller on local host
```
$ kind create cluster
$ make apply-manifests
$ make build
$ make run
```

## Running controller in Docker
```
$ kind create cluster
$ make apply-manifests
$ make build-image
$ make port-forward &
$ make push-image
```

## See also
* [sample-controller](https://github.com/kubernetes/sample-controller)
* [kubebuilder](https://github.com/kubernetes-sigs/kubebuilder)
* [operator-sdk](https://github.com/operator-framework/operator-sdk)
* [kind](https://github.com/kubernetes-sigs/kind)
* [Kubernetes Reference](https://kubernetes.io/docs/reference/)

## TODO
You can edit the following files as needed.

```
$ grep -riIl --exclude-dir=generated --exclude-dir=.git --exclude=zz_generated.deepcopy.go 'supercaracal\|foobar\|kubernetes-controller-template' .
./README.md
./go.mod
./.github/workflows/release.yaml
./internal/controller/custom.go
./internal/worker/reconciler.go
./Makefile
./.dockerignore
./.gitignore
./config/controller.yaml
./config/registry.yaml
./config/crd.yaml
./config/example-foobar.yaml
./main.go
./pkg/apis/supercaracal/register.go
./pkg/apis/supercaracal/v1/doc.go
./pkg/apis/supercaracal/v1/register.go
./pkg/apis/supercaracal/v1/types.go
./Dockerfile
```

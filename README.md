![](https://github.com/supercaracal/aws-ecr-image-pull-secret-controller/workflows/Test/badge.svg?branch=master)
![](https://github.com/supercaracal/aws-ecr-image-pull-secret-controller/workflows/Release/badge.svg)

AWS ECR image pull secret controller
===============================================================================

This controller has a feature to renew image-pull secrets for AWS ECR.
Since docker login for AWS ECR expires at 12 hours later, the controller is needed for non EKS.

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

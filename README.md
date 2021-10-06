![](https://github.com/supercaracal/aws-ecr-image-pull-secret-controller/workflows/Test/badge.svg?branch=master)
![](https://github.com/supercaracal/aws-ecr-image-pull-secret-controller/workflows/Release/badge.svg)

AWS ECR image pull secret controller
===============================================================================

This controller has a feature to renew image-pull secrets for AWS ECR.
Since docker login for AWS ECR expires at 12 hours later, the controller is needed for non EKS.

## Controller's action
This controller checks all image pull secrets every 10 second.
The controller acts as the followings.

| image pull secret | expiration | action |
| --- | --- | --- |
| not exists | | creates a new image pull secret |
| exists | valid | does nothing |
| exists | expired | deletes old image pull secret and creates new one |

## Running controller on local host
```
$ kind create cluster
$ make apply-manifests
$ make build
$ make run
```

## Running controller in cluster
```
$ kind create cluster
$ make apply-manifests
$ make build-image
$ make port-forward &
$ make push-image
```

## Usage
```yaml
apiVersion: v1
kind: Secret
metadata:
  name: example-login-secret
  labels:
    supercaracal.example.com/used-by: "aws-ecr-image-pull-secret-controller"
  annotations:
    supercaracal.example.com/aws-ecr-image-pull-secret.name: "example-image-pull-secret"
    supercaracal.example.com/aws-ecr-image-pull-secret.email: "foobar@example.com"
    supercaracal.example.com/aws-ecr-image-pull-secret.aws_account_id: "000000000000"
    supercaracal.example.com/aws-ecr-image-pull-secret.aws_region: "ap-northeast-1"
type: Opaque
data:
  AWS_ACCESS_KEY_ID: "**********base64 encoded text**********"
  AWS_SECRET_ACCESS_KEY: "**********base64 encoded text**********"
```

```
$ cp config/example-secret.yaml config/secret.yaml
$ vi config/secret.yaml
$ kubectl --context=kind-kind apply -f config/secret.yaml
```

```
$ kubectl --context=kind-kind get secrets
NAME                        TYPE                                  DATA   AGE
controller-token-8bmfl      kubernetes.io/service-account-token   3      37m
default-token-s4wsj         kubernetes.io/service-account-token   3      39m
example-image-pull-secret   kubernetes.io/dockerconfigjson        1      10m
example-login-secret        Opaque                                2      33m
```

```
$ kubectl --context=kind-kind get secrets example-image-pull-secret -o json | jq -r .data.'".dockerconfigjson"' | base64 -d | jq .
{
  "auths": {
    "https://000000000000.dkr.ecr.ap-northeast-1.amazonaws.com": {
      "username": "AWS",
      "password": "*****************************************",
      "email": "foo@example.com",
      "auth": "*****************************************"
    }
  }
}
```

## See also
* [sample-controller](https://github.com/kubernetes/sample-controller)
* [kubebuilder](https://github.com/kubernetes-sigs/kubebuilder)
* [operator-sdk](https://github.com/operator-framework/operator-sdk)
* [kind](https://github.com/kubernetes-sigs/kind)
* [Kubernetes Reference](https://kubernetes.io/docs/reference/)
* [Configure a kubelet image credential provider](https://kubernetes.io/docs/tasks/kubelet-credential-provider/kubelet-credential-provider/)

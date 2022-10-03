MAKEFLAGS += --warn-undefined-variables
SHELL     := /bin/bash -euo pipefail
SVC       := github.com
ORG       := supercaracal
REPO      := aws-ecr-image-pull-secret-controller
MOD_PATH  := ${SVC}/${ORG}/${REPO}
IMG_TAG   := latest
REGISTRY  := 127.0.0.1:5000
GOBIN     ?= $(shell go env GOPATH)/bin

ifdef VERBOSE
	QUIET :=
else
	QUIET := @
endif

all: build test lint

${GOBIN}/golint:
	go install golang.org/x/lint/golint@latest

build: GOOS        ?= $(shell go env GOOS)
build: GOARCH      ?= $(shell go env GOARCH)
build: CGO_ENABLED ?= $(shell go env CGO_ENABLED)
build: FLAGS       += -ldflags="-s -w"
build: FLAGS       += -trimpath
build: FLAGS       += -tags timetzdata
build:
	GOOS=${GOOS} GOARCH=${GOARCH} CGO_ENABLED=${CGO_ENABLED} go build ${FLAGS} -o ${REPO}

test:
	${QUIET} go clean -testcache
	${QUIET} go test -race ./...

lint: ${GOBIN}/golint
	${QUIET} go vet ./...
	${QUIET} golint -set_exit_status ./...

run: TZ  ?= Asia/Tokyo
run: CFG ?= $$HOME/.kube/config
run:
	${QUIET} TZ=${TZ} ./${REPO} --kubeconfig=${CFG}

clean:
	${QUIET} rm -rf ${REPO} main

build-image:
	${QUIET} docker build -t ${REPO}:${IMG_TAG} .

lint-image:
	${QUIET} docker run --rm -i hadolint/hadolint < Dockerfile

port-forward:
	${QUIET} kubectl --context=kind-kind port-forward service/registry 5000:5000

push-image:
	${QUIET} docker tag ${REPO}:${IMG_TAG} ${REGISTRY}/${REPO}:${IMG_TAG}
	${QUIET} docker push ${REGISTRY}/${REPO}:${IMG_TAG}

clean-image:
	${QUIET} docker rmi -f ${REPO}:${IMG_TAG} ${REGISTRY}/${REPO}:${IMG_TAG} || true
	${QUIET} docker image prune -f
	${QUIET} docker volume prune -f

apply-manifests:
	${QUIET} kubectl --context=kind-kind apply -f config/registry.yaml
	${QUIET} kubectl --context=kind-kind apply -f config/controller.yaml

replace-k8s-go-module: KUBE_LIB_VER := 1.25.2
replace-k8s-go-module:
	${QUIET} ./scripts/replace_k8s_go_module.sh ${KUBE_LIB_VER}

wait-registry-running:
	${QUIET} ./scripts/wait_pod_status.sh registry Running

wait-controller-running:
	${QUIET} ./scripts/wait_pod_status.sh controller Running

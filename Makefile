MAKEFLAGS += --warn-undefined-variables
SHELL     := /bin/bash -euo pipefail
SVC       := github.com
ORG       := supercaracal
REPO      := kubernetes-controller-template
MOD_PATH  := ${SVC}/${ORG}/${REPO}
IMG_TAG   := latest
REGISTRY  := 127.0.0.1:5000
TEMP_DIR  := _tmp
GOBIN     ?= $(shell go env GOPATH)/bin

ifdef VERBOSE
	QUIET :=
else
	QUIET := @
endif

all: build test lint

${TEMP_DIR}:
	${QUIET} mkdir -p $@

${GOBIN}/deepcopy-gen ${GOBIN}/client-gen ${GOBIN}/lister-gen ${GOBIN}/informer-gen:
	go install k8s.io/code-generator/...@latest

${GOBIN}/golint:
	go install golang.org/x/lint/golint@latest

# https://github.com/kubernetes/gengo/blob/master/args/args.go
# https://github.com/kubernetes/code-generator/tree/master/cmd
${TEMP_DIR}/codegen: GOENV                 += GOROOT=${CURDIR}/${TEMP_DIR}
${TEMP_DIR}/codegen: LOG_LEVEL             ?= 1
${TEMP_DIR}/codegen: API_VERSION           := v1
${TEMP_DIR}/codegen: CODE_GEN_INPUT        := ${MOD_PATH}/pkg/apis/${ORG}/${API_VERSION}
${TEMP_DIR}/codegen: CODE_GEN_OUTPUT       := ${MOD_PATH}/pkg/generated
${TEMP_DIR}/codegen: CODE_GEN_ARGS         += --output-base=${CURDIR}/${TEMP_DIR}/src
${TEMP_DIR}/codegen: CODE_GEN_ARGS         += --go-header-file=${CURDIR}/${TEMP_DIR}/empty.txt
${TEMP_DIR}/codegen: CODE_GEN_ARGS         += -v ${LOG_LEVEL}
${TEMP_DIR}/codegen: CODE_GEN_DEEPC        := zz_generated.deepcopy
${TEMP_DIR}/codegen: CODE_GEN_CLI_SET_NAME := versioned
${TEMP_DIR}/codegen: ${GOBIN}/deepcopy-gen ${GOBIN}/client-gen ${GOBIN}/lister-gen ${GOBIN}/informer-gen ${TEMP_DIR} $(shell find pkg/apis/${ORG}/ -type f -name '*.go')
	${QUIET} touch -a ${TEMP_DIR}/empty.txt
	${QUIET} mkdir -p ${TEMP_DIR}/src/${MOD_PATH}
	${QUIET} ln -sf ${CURDIR}/pkg ${TEMP_DIR}/src/${MOD_PATH}/
	${GOENV} ${GOBIN}/deepcopy-gen ${CODE_GEN_ARGS} --input-dirs=${CODE_GEN_INPUT} --bounding-dirs=${CODE_GEN_INPUT} --output-file-base=${CODE_GEN_DEEPC}
	${GOENV} ${GOBIN}/client-gen   ${CODE_GEN_ARGS} --input=${CODE_GEN_INPUT}      --output-package=${CODE_GEN_OUTPUT}/clientset --input-base="" --clientset-name=${CODE_GEN_CLI_SET_NAME}
	${GOENV} ${GOBIN}/lister-gen   ${CODE_GEN_ARGS} --input-dirs=${CODE_GEN_INPUT} --output-package=${CODE_GEN_OUTPUT}/listers
	${GOENV} ${GOBIN}/informer-gen ${CODE_GEN_ARGS} --input-dirs=${CODE_GEN_INPUT} --output-package=${CODE_GEN_OUTPUT}/informers --versioned-clientset-package=${CODE_GEN_OUTPUT}/clientset/${CODE_GEN_CLI_SET_NAME} --listers-package=${CODE_GEN_OUTPUT}/listers
	${QUIET} touch $@

codegen: ${TEMP_DIR}/codegen

build: GOOS        ?= $(shell go env GOOS)
build: GOARCH      ?= $(shell go env GOARCH)
build: CGO_ENABLED ?= $(shell go env CGO_ENABLED)
build: FLAGS       += -ldflags="-s -w"
build: FLAGS       += -trimpath
build: FLAGS       += -tags timetzdata
build: codegen
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
	${QUIET} unlink ${TEMP_DIR}/src/${MOD_PATH}/pkg || true
	${QUIET} rm -rf ${REPO} main ${TEMP_DIR} pkg/generated pkg/apis/*/*/zz_generated.deepcopy.go

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
	${QUIET} kubectl --context=kind-kind apply -f config/crd.yaml
	${QUIET} kubectl --context=kind-kind apply -f config/example-foobar.yaml
	${QUIET} kubectl --context=kind-kind apply -f config/controller.yaml

replace-k8s-go-module: KUBE_LIB_VER := 1.22.1
replace-k8s-go-module:
	${QUIET} ./scripts/replace_k8s_go_module.sh ${KUBE_LIB_VER}

wait-registry-running:
	${QUIET} ./scripts/wait_pod_status.sh registry Running

wait-controller-running:
	${QUIET} ./scripts/wait_pod_status.sh controller Running

wait-example-completed:
	${QUIET} ./scripts/wait_pod_status.sh example Succeeded

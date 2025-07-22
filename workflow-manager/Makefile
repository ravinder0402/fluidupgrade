VERSION ?= latest
REPO ?= coredgeio
ROOT_DIR := $(shell dirname $(realpath $(lastword $(MAKEFILE_LIST))))

HELM := helm
HELM_REGISTRY ?= oci://registry-1.docker.io/coredgehelm
HELM_DIR := $(ROOT_DIR)/helm
PACKAGE_DIR := $(HELM_DIR)/dist

GIT_TOKEN ?= ""

IMG = $(REPO)/workflow-manager:$(VERSION)
INIT_JOB = $(REPO)/workflow-init:$(VERSION)

.PHONY: all

all: build-manager build-init

build-manager:
	go fmt ./...
	go vet ./...
ifeq ($(GIT_TOKEN),"")
	rsync -rupE $$HOME/.ssh/* .ssh/
	sudo docker build -t ${IMG} -f build/manager/Dockerfile.ssh .
else
	sudo docker build --build-arg GIT_TOKEN="${GIT_TOKEN}" \
		-t ${IMG} -f build/manager/Dockerfile .
endif

build-init:
	go fmt ./...
	go vet ./...
ifeq ($(GIT_TOKEN),"")
	rsync -rupE $$HOME/.ssh/* .ssh/
	sudo docker build -t ${INIT_JOB} -f build/init/Dockerfile.ssh .
else
	sudo docker build --build-arg GIT_TOKEN="${GIT_TOKEN}" \
		-t ${INIT_JOB} -f build/init/Dockerfile .
endif

push-images:
	sudo docker push ${IMG}
	sudo docker push ${INIT_JOB}

cleanup-helm:
	rm -rf ${PACKAGE_DIR}/*
	rm -rf ${HELM_DIR}/charts
	rm -rf ${HELM_DIR}/*.lock

build-helm: cleanup-helm
	${HELM} dep up ${HELM_DIR}
	${HELM} lint ${HELM_DIR}
	${HELM} package -d ${PACKAGE_DIR} ${HELM_DIR}

push-helm:
	${HELM} push ${PACKAGE_DIR}/*.tgz ${HELM_REGISTRY}

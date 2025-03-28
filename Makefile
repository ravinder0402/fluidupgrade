HELM := helm
HELM_REGISTRY ?= oci://registry-1.docker.io/coredgehelm
ROOT_DIR := $(shell dirname $(realpath $(lastword $(MAKEFILE_LIST))))
PACKAGE_DIR := $(ROOT_DIR)/dist

HELM_CHARTS := helm

BUILD_CHARTS := $(foreach chart, $(HELM_CHARTS), build-$(chart))

PUSH_HELM_CHARTS := helm

.phony: all

all: $(BUILD_CHARTS)

.ONESHELL:
build-%: lint-% init-%
	@echo "========================================="
	@echo "      helm pack	$*     "
	@echo "========================================="
	@mkdir -p $(PACKAGE_DIR)
	if [ -f $*/Chart.yaml ]
	then
		$(HELM) package -d $(PACKAGE_DIR) $*
		$(HELM) repo index $(PACKAGE_DIR)
	fi

lint-%: init-%
	@echo "===================================="
	@echo "      helm lint	$*     "
	@echo "===================================="
	if [ -f $*/Chart.yaml ]; then $(HELM) lint $*; fi

init-%:
	@echo "============================================="
	@echo " helm dependency update	$* "
	@echo "============================================="
	if [ -f $*/Chart.yaml ]; then $(HELM) dep up $*; fi

push-helm:
	@echo "============================================="
	@echo " pushing helm chart $* "
	@echo "============================================="
	if [ -f $(PACKAGE_DIR)/$**.tgz ];
	then
		helm push $(PACKAGE_DIR)/$**.tgz $(HELM_REGISTRY)
	fi

clean:
	rm -rf dist
	rm -rf */charts
	rm -rf */*.lock

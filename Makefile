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
	@echo " Pushing helm charts "
	@echo "============================================="
	@for chart in $(PUSH_HELM_CHARTS); do \
		tgz_file=$$(ls $(PACKAGE_DIR)/$$chart-*.tgz 2>/dev/null | head -n 1); \
		if [ -n "$$tgz_file" ]; then \
			version=$$(basename $$tgz_file | sed 's/^.*-\(.*\)\.tgz/\1/'); \
			echo "Pushing $$tgz_file"; \
			export HELM_EXPERIMENTAL_OCI=1; \
			$(HELM) push $$tgz_file $(HELM_REGISTRY); \
		else \
			echo "No .tgz file found for $$chart in $(PACKAGE_DIR)"; \
		fi \
	done

clean:
	rm -rf dist
	rm -rf */charts
	rm -rf */*.lock

# Copyright (c) 2020, 2021, Oracle and/or its affiliates.
# Licensed under the Universal Permissive License v 1.0 as shown at https://oss.oracle.com/licenses/upl.

include make/quality.mk

SCRIPT_DIR:=$(shell dirname $(realpath $(firstword $(MAKEFILE_LIST))))/build
ROOT_DIR:=$(shell dirname $(realpath $(firstword $(MAKEFILE_LIST))))
TOOLS_DIR:=$(shell dirname $(realpath $(firstword $(MAKEFILE_LIST))))/tools

ifeq ($(MAKECMDGOALS),$(filter $(MAKECMDGOALS),docker-push create-test-deploy))
ifndef DOCKER_REPO
    $(error DOCKER_REPO must be defined as the name of the docker repository where image will be pushed)
endif
ifndef DOCKER_NAMESPACE
    $(error DOCKER_NAMESPACE must be defined as the name of the docker namespace where image will be pushed)
endif
endif

TIMESTAMP := $(shell date -u +%Y%m%d%H%M%S)
DOCKER_IMAGE_TAG ?= local-${TIMESTAMP}-$(shell git rev-parse --short HEAD)
VERRAZZANO_PLATFORM_OPERATOR_IMAGE_NAME ?= verrazzano-platform-operator-dev
VERRAZZANO_APPLICATION_OPERATOR_IMAGE_NAME ?= verrazzano-application-operator-dev
VERRAZZANO_ANALYSIS_IMAGE_NAME ?= verrazzano-analysis-dev
VERRAZZANO_IMAGE_PATCH_OPERATOR_IMAGE_NAME ?= verrazzano-image-patch-operator-dev
VERRAZZANO_WEBLOGIC_IMAGE_TOOL_IMAGE_NAME ?= verrazzano-weblogic-image-tool-dev

VERRAZZANO_PLATFORM_OPERATOR_IMAGE = ${DOCKER_REPO}/${DOCKER_NAMESPACE}/${VERRAZZANO_PLATFORM_OPERATOR_IMAGE_NAME}:${DOCKER_IMAGE_TAG}
VERRAZZANO_APPLICATION_OPERATOR_IMAGE = ${DOCKER_REPO}/${DOCKER_NAMESPACE}/${VERRAZZANO_APPLICATION_OPERATOR_IMAGE_NAME}:${DOCKER_IMAGE_TAG}
VERRAZZANO_ANALYSIS_IMAGE = ${DOCKER_REPO}/${DOCKER_NAMESPACE}/${VERRAZZANO_ANALYSIS_IMAGE_NAME}:${DOCKER_IMAGE_TAG}
VERRAZZANO_IMAGE_PATCH_OPERATOR_IMAGE = ${DOCKER_REPO}/${DOCKER_NAMESPACE}/${VERRAZZANO_IMAGE_PATCH_OPERATOR_IMAGE_NAME}:${DOCKER_IMAGE_TAG}

CURRENT_YEAR = $(shell date +"%Y")

PARENT_BRANCH ?= origin/master

GO ?= CGO_ENABLED=0 GO111MODULE=on GOPRIVATE=github.com/verrazzano go
GO_LDFLAGS ?= -extldflags -static -X main.buildVersion=${BUILDVERSION} -X main.buildDate=${BUILDDATE}

.PHONY: clean
clean:
	find . -name coverage.cov -exec rm {} \;
	find . -name coverage.html -exec rm {} \;
	find . -name coverage.raw.cov -exec rm {} \;
	find . -name \*-test-result.xml -exec rm {} \;

.PHONY: docker-push
docker-push:
	(cd application-operator; make docker-push DOCKER_IMAGE_NAME=${VERRAZZANO_APPLICATION_OPERATOR_IMAGE_NAME} DOCKER_IMAGE_TAG=${DOCKER_IMAGE_TAG})
	(cd platform-operator; make docker-push DOCKER_IMAGE_NAME=${VERRAZZANO_PLATFORM_OPERATOR_IMAGE_NAME} DOCKER_IMAGE_TAG=${DOCKER_IMAGE_TAG} VERRAZZANO_APPLICATION_OPERATOR_IMAGE=${VERRAZZANO_APPLICATION_OPERATOR_IMAGE})
	(cd tools/analysis; make docker-push DOCKER_IMAGE_NAME=${VERRAZZANO_ANALYSIS_IMAGE_NAME} DOCKER_IMAGE_TAG=${DOCKER_IMAGE_TAG})

.PHONY: docker-push-ipo
docker-push-ipo:
	(cd image-patch-operator; make docker-push DOCKER_IMAGE_NAME=${VERRAZZANO_IMAGE_PATCH_OPERATOR_IMAGE_NAME} DOCKER_IMAGE_TAG=${DOCKER_IMAGE_TAG})

.PHONY: docker-push-wit
docker-push-wit:
	(cd image-patch-operator/weblogic-imagetool; make docker-push DOCKER_IMAGE_NAME=${VERRAZZANO_WEBLOGIC_IMAGE_TOOL_IMAGE_NAME} DOCKER_IMAGE_TAG=${DOCKER_IMAGE_TAG})

.PHONY: create-test-deploy
create-test-deploy: docker-push
	(cd platform-operator; make create-test-deploy VZ_DEV_IMAGE=${VERRAZZANO_PLATFORM_OPERATOR_IMAGE} VZ_APP_OP_IMAGE=${VERRAZZANO_APPLICATION_OPERATOR_IMAGE})

.PHONY: test-platform-operator-install
test-platform-operator-install:
	kubectl apply -f platform-operator/build/deploy/operator.yaml
	kubectl -n verrazzano-install rollout status deployment/verrazzano-platform-operator

.PHONY: test-platform-operator-remove
test-platform-operator-remove:
	kubectl delete -f platform-operator/build/deploy/operator.yaml

.PHONY: test-platform-operator-install-logs
test-platform-operator-install-logs:
	kubectl logs -f -n default $(shell kubectl get pods -n default --no-headers | grep "^verrazzano-install-" | cut -d ' ' -f 1)

#
#  Compliance check targets
#

.PHONY: copyright-test
copyright-test:
	(cd tools/copyright; go test .)

.PHONY: copyright-check-year
copyright-check-year: copyright-test
	go run tools/copyright/copyright.go --enforce-current $(shell git log --since=01-01-${CURRENT_YEAR} --name-only --oneline --pretty="format:" | sort -u)

.PHONY: copyright-check
copyright-check: copyright-test
	go run tools/copyright/copyright.go --verbose --enforce-current .

.PHONY: copyright-check-local
copyright-check-local: copyright-test
	go run tools/copyright/copyright.go --verbose --enforce-current  $(shell git status --short | cut -c 4-)

.PHONY: copyright-check-branch
copyright-check-branch: copyright-check
	go run tools/copyright/copyright.go --verbose --enforce-current $(shell git diff --name-only ${PARENT_BRANCH})

#
# Quality checks on acceptance tests
#
.PHONY: check-tests
check-tests: check-eventually

.PHONY: check-eventually
check-eventually: check-eventually-test
	go run tools/eventually-checker/check_eventually.go tests/e2e

.PHONY: check-eventually-test
check-eventually-test:
	(cd tools/eventually-checker; go test .)

#
# CLI
#
cli:
	(cd tools/cli/vz; go install)

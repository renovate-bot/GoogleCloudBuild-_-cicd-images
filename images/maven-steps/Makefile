REGISTRY ?= us-docker.pkg.dev/gcb-catalog-release/catalog
TAG ?= deprecated-public-image-ubuntu24

build:
	@echo "+ building maven-steps image"
	cd .. && docker buildx bake maven-steps

test:
	container-structure-test test --image ${REGISTRY}/maven-steps:${TAG} --config test_config.yaml

.PHONY: build test
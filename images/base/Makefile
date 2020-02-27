TAG?=$(shell echo "$$(date +v%Y%m%d)-$$(git describe --always --dirty)")
IMAGE?=kindest/base:$(TAG)

# build with buildx
PLATFORMS?=linux/amd64,linux/arm64
OUTPUT=
build: ensure-buildx
	docker buildx build --platform=${PLATFORMS} $(OUTPUT) -t ${IMAGE} --pull .

# push the cross built image
push: OUTPUT=--push
push: build

# quick can be used to do a build that will be imported into the local docker 
# for sanity checking before doing a cross build push
# cross builds cannot be imported locally at the moment
# https://github.com/docker/buildx/issues/59
quick: PLATFORMS=linux/amd64
quick: OUTPUT=--load
quick: build

# enable buildx
DOCKER_CLI_EXPERIMENTAL=enabled
ensure-buildx:
	./../../hack/build/init-buildx.sh

.PHONY: push build quick ensure-buildx
GIT_COMMIT=$(shell git log -1 --format='%h' | awk '{print $0}')
DATE=$(shell date +%Y-%m-%d_%H:%M:%S%z)
GO_VERSION=$(shell go version)

DOCKER_REGISTRY ?= public.ecr.aws
DOCKER_REPO ?= ${DOCKER_REGISTRY}/vanus
IMAGE_TAG ?= ${GIT_COMMIT}
#os linux or darwin
GOOS ?= linux
#arch amd64 or arm64
GOARCH ?= amd64

VERSION ?= ${IMAGE_TAG}

GO_BUILD= GO111MODULE=on CGO_ENABLED=0 GOOS=$(GOOS) GOARCH=$(GOARCH) go build -trimpath
DOCKER_BUILD_ARG= --build-arg TARGETARCH=$(GOARCH) --build-arg TARGETOS=$(GOOS)
DOCKER_PLATFORM ?= linux/amd64

docker-push:
	docker buildx build --platform ${DOCKER_PLATFORM} \
		-t ${DOCKER_REPO}/gateway-proxy:${IMAGE_TAG} \
		-f Dockerfile . --push

docker-build:
	docker buildx build --platform ${DOCKER_PLATFORM} \
		-t ${DOCKER_REPO}/gateway-proxy:${IMAGE_TAG} \
		-f Dockerfile . --load

build:
	$(GO_BUILD)  -o bin/proxy cmd/main.go
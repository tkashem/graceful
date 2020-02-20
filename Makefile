MOD_FLAGS := $(shell (go version | grep -q -E "1\.1[1-9]") && echo -mod=vendor)
CMD_PACKAGE := $(shell go list $(MOD_FLAGS) ./cmd/...)
OUTPUT_DIR := "_output/bin"
BINARY := "$(OUTPUT_DIR)/graceful-test"

IMAGE ?= "docker.io/tohinkashem/graceful:latest"

build:
	mkdir -p ./_output/bin
	go build $(MOD_FLAGS) -o $(BINARY) $(CMD_PACKAGE)

image:
	docker build -t $(IMAGE) -f Dockerfile .
	docker push $(IMAGE)


push: build image

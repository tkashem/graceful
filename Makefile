MOD_FLAGS := $(shell (go version | grep -q -E "1\.1[1-9]") && echo -mod=vendor)
CMDS := $(shell go list $(MOD_FLAGS) ./cmd/...)
OUTPUT_DIR := "_output/bin"

IMAGE ?= "docker.io/tohinkashem/graceful:latest"

$(CMDS):
	go build $(MOD_FLAGS) -o $(OUTPUT_DIR)/$(shell basename $@) $@

build: clean $(CMDS)

clean:
	rm -rf ./_output/bin
	mkdir -p ./_output/bin


#	go build $(MOD_FLAGS) -o $(BINARY) $(CMD_PACKAGE)

image:
	docker build -t $(IMAGE) -f Dockerfile .
	docker push $(IMAGE)

prometheus: prometheus-build prometheus-run

prometheus-build:
	docker build -t myprometheus -f prometheus/Dockerfile.prometheus .

prometheus-run:
	echo "killing prometheus instance" >&2
	docker kill prometheus 2>/dev/null || true
	docker run --rm -d --network=host --name=prometheus myprometheus



push: build image

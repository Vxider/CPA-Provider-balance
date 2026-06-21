PLUGIN_NAME := provider-balance
OUT_DIR := bin
EXT := so
LOCAL_GOPATH ?= /home/vxider/.go
GOENV := GOPATH=$(LOCAL_GOPATH) GOFLAGS=-mod=mod

.PHONY: build test clean install

build:
	mkdir -p $(OUT_DIR)
	$(GOENV) go build -buildvcs=false -buildmode=c-shared -o $(OUT_DIR)/$(PLUGIN_NAME).$(EXT) ./cmd/provider-balance

test:
	$(GOENV) go test ./...

install: build
	mkdir -p /home/vxider/cliproxyapi/plugins/linux/arm64
	cp $(OUT_DIR)/$(PLUGIN_NAME).$(EXT) /home/vxider/cliproxyapi/plugins/linux/arm64/$(PLUGIN_NAME).so

clean:
	rm -rf $(OUT_DIR)

# library versions
PROTOBUF_VERSION := 3.15.8

# paths
ROOT_PATH := $(dir $(abspath $(lastword $(MAKEFILE_LIST))))
BIN_PATH := $(ROOT_PATH)/bin
WORK_PATH ?= /tmp/oinari-lib-go-work

.PHONY: build
build: api/oinari_grpc.pb.go internal/fox/fox.glb
	go build -o $(BIN_PATH)/fox cmd/fox/native/*.go
	GOOS=js GOARCH=wasm go build -o $(BIN_PATH)/fox.wasm cmd/fox/wasm/*.go

api/oinari_grpc.pb.go: api/oinari.proto
	PATH="${PATH}:$(shell go env GOPATH)/bin" $(BIN_PATH)/protoc --go_out=. --go_opt=module=github.com/llamerada-jp/oinari-lib-go --go-grpc_out=. --go-grpc_opt=module=github.com/llamerada-jp/oinari-lib-go api/oinari.proto

internal/fox/fox.glb:
	curl -Lo internal/fox/fox.glb https://github.com/KhronosGroup/glTF-Sample-Models/raw/master/2.0/Fox/glTF-Binary/Fox.glb

.PHONY: clean
clean:
	rm -rf $(WORK_PATH)

.PHONY: setup
setup:
	rm -rf $(WORK_PATH)
	mkdir -p $(BIN_PATH)
	git clone https://github.com/llamerada-jp/colonio-go.git $(WORK_PATH)/colonio-go
	curl -Lo $(WORK_PATH)/protobuf.zip https://github.com/protocolbuffers/protobuf/releases/download/v$(PROTOBUF_VERSION)/protoc-$(PROTOBUF_VERSION)-linux-x86_64.zip
	unzip $(WORK_PATH)/protobuf.zip -d $(WORK_PATH)/protobuf
	cp -a $(WORK_PATH)/protobuf/bin/protoc $(BIN_PATH)/protoc
	go get google.golang.org/protobuf/cmd/protoc-gen-go google.golang.org/grpc/cmd/protoc-gen-go-grpc

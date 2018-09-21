BUILD_DIR?=$(shell pwd)/build
GOX_OS=
GOX_OSARCH=darwin/amd64 linux/amd64 windows/amd64

.PHONY: format
format:
	gofmt -d -w -s -e ./bin/ ./config/

.PHONY: test
test:
	go test ./... -v

.PHONY: deps
deps:
	go get github.com/mitchellh/gox

.PHONY: crosscompile
crosscompile: deps
	mkdir -p ${BUILD_DIR}/bin
	gox -output="${BUILD_DIR}/bin/{{.Dir}}-{{.OS}}-{{.Arch}}" -os="$(strip $(GOX_OS))" -osarch="$(strip $(GOX_OSARCH))" ${GOX_FLAGS}

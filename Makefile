GOOS ?= linux

.PHONY: build
build:
	GOOS=${GOOS} GOARCH=amd64 CGO_ENABLED=0 go build -o bin/rf -trimpath -ldflags="-w -s"

.PHONY: test
test:
	go test ./... -v

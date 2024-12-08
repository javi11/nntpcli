APP ?= usenetdrive
CGO_ENABLED ?= 1
GOOS ?= $(shell go env GOOS)
GOARCH ?= $(shell go env GOARCH)
DIST_PATH ?= dist
BUILD_PATH ?= $(DIST_PATH)/$(APP)_$(GOOS)_$(GOARCH)
GIT_COMMIT ?= $(shell git rev-parse --short HEAD)
TIMESTAMP ?= $(shell date +%s)
VERSION ?= 0.0.0-dev

.DEFAULT_GOAL := test

.PHONY: tools
tools:
	go install github.com/golangci/golangci-lint/cmd/golangci-lint@latest
	go install github.com/goreleaser/goreleaser@latest
	go install github.com/golang/mock/mockgen@v1.6.0
	go install golang.org/x/vuln/cmd/govulncheck@latest
	
.PHONY: lint
lint: tools
	golangci-lint run --timeout 5m

.PHONY: generate
generate: tools
	go generate ./...

.PHONY: test
test: generate lint
	go test ./... -cover -v -race ${GO_PACKAGES}
	
.PHONY: release
release:
	goreleaser --skip-validate --skip-publish --rm-dist

.PHONY: snapshot
snapshot: build_web
	goreleaser --snapshot --skip-publish --rm-dist

.PHONY: publish
publish:
	goreleaser --rm-dist
	
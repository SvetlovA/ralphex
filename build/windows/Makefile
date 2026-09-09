# Windows Makefile — invoke from project root:
#   make -C build/windows <target>
#   make -f build/windows/Makefile <target>
# Requires: Go, git, golangci-lint (for lint), goimports (for fmt)
SHELL := cmd.exe

# auto-detect project root (works with both -f and -C invocation)
ROOT := $(dir $(lastword $(MAKEFILE_LIST)))../..

# Get the latest commit branch, hash, and date
TAG=$(shell git describe --tags --abbrev=0 --exact-match 2>NUL)
BRANCH=$(if $(TAG),$(TAG),$(shell git rev-parse --abbrev-ref HEAD 2>NUL))
HASH=$(shell git rev-parse --short=7 HEAD 2>NUL)
TIMESTAMP=$(shell git log -1 --format=%%ct HEAD 2>NUL)
GIT_REV=$(BRANCH)-$(HASH)-$(TIMESTAMP)
REV=$(if $(filter --,$(GIT_REV)),latest,$(GIT_REV))

all: test build

build:
	cd $(ROOT)/cmd/ralphex && go build -ldflags "-X main.revision=$(REV) -s -w" -o ../../.bin/ralphex.$(BRANCH).exe
	cd $(ROOT) && copy /Y .bin\ralphex.$(BRANCH).exe .bin\ralphex.exe

test:
	cd $(ROOT) && go clean -testcache
	cd $(ROOT) && go test -coverprofile=coverage.out ./...
	cd $(ROOT) && findstr /V _mock.go coverage.out | findstr /V mocks > coverage_no_mocks.out
	cd $(ROOT) && go tool cover -func=coverage_no_mocks.out
	cd $(ROOT) && del /q coverage.out coverage_no_mocks.out

lint:
	cd $(ROOT) && golangci-lint run --max-issues-per-linter=0 --max-same-issues=0

fmt:
	cd $(ROOT) && gofmt -s -w cmd pkg e2e
	cd $(ROOT) && goimports -w cmd pkg e2e

race:
	cd $(ROOT) && go test -race -timeout=60s ./...

version:
	@echo branch: $(BRANCH), hash: $(HASH), timestamp: $(TIMESTAMP)
	@echo revision: $(REV)

e2e-setup:
	cd $(ROOT) && go run github.com/playwright-community/playwright-go/cmd/playwright@latest install --with-deps chromium

e2e:
	cd $(ROOT) && go test -v -failfast -count=1 -timeout=5m -tags=e2e ./e2e/...

.PHONY: all build test lint fmt race version e2e-setup e2e

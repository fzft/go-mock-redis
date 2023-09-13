GIT_SHA1 := $(shell git rev-parse HEAD || echo "unknown")
GIT_DIRTY := $(shell git diff --quiet || echo "dirty")
BUILD_ID := $(shell uname -n)-$(shell date +%s)
BUILD_DATE := $(shell date -u '+%Y-%m-%d')

build:
	go build -ldflags="-X main.gitSHA1=$(GIT_SHA1) -X main.gitDirty=$(GIT_DIRTY) -X main.buildID=$(BUILD_ID) -X main.buildDate=$(BUILD_DATE)"

test:
	go test -v ./...

.PHONY: build, test

VERSION := 0.1.0
NAME := bruya
GH_PATH := github.com/kung-foo/$(NAME)
BUILDSTRING := $(shell git log --pretty=format:'%h' -n 1)
VERSIONSTRING := $(NAME) version $(VERSION)+$(BUILDSTRING)
OUTPUT = dist/$(NAME)

GO_VER := 1.10.3

UID := $(shell id -u)
GID := $(shell id -g)

DOCKER_OPTS := -v "$(PWD)":/go/src/$(GH_PATH) -w /go/src/$(GH_PATH)

BUILD_CMD := go build -o $(OUTPUT) -ldflags "-X \"main.VERSION=$(VERSIONSTRING)\"" app/main.go
RUN_CMD := docker run --rm -u $(UID):$(GID) $(DOCKER_OPTS)

clean:
	rm -rf dist/*

$(OUTPUT): Gopkg.lock app/main.go *.go
	@mkdir -p dist/
	$(BUILD_CMD)

build-docker-alpine:
	$(RUN_CMD) golang:$(GO_VER)-alpine $(BUILD_CMD)
	mv $(OUTPUT) $(OUTPUT)-musl

build-docker-debian:
	$(RUN_CMD)  golang:$(GO_VER) $(BUILD_CMD)
	mv $(OUTPUT) $(OUTPUT)-debian

build-docker-windows:
	$(RUN_CMD) -e GOOS=windows golang:$(GO_VER) $(BUILD_CMD)
	mv $(OUTPUT) $(OUTPUT).exe

release: build-docker-alpine build-docker-alpine build-docker-windows

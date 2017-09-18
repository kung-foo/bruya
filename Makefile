VERSION := 0.0.0
NAME := bruya
GH_PATH := github.com/kung-foo/$(NAME)
BUILDSTRING := $(shell git log --pretty=format:'%h' -n 1)
VERSIONSTRING := $(NAME) version $(VERSION)+$(BUILDSTRING)
OUTPUT = dist/$(NAME)
BUILD_CMD := go build -o $(OUTPUT) -ldflags "-X \"main.VERSION=$(VERSIONSTRING)\"" app/main.go
UID := $(shell id -u)
GID := $(shell id -g)

clean:
	rm -rf dist/*

$(OUTPUT): glide.lock app/main.go *.go
	@mkdir -p dist/
	$(BUILD_CMD)

DOCKER_OPTS := -v "$(PWD)":/go/src/$(GH_PATH) -w /go/src/$(GH_PATH)

build-docker-alpine:
	docker run --rm -u $(UID):$(GID) $(DOCKER_OPTS) golang:1.9-alpine $(BUILD_CMD)
	mv $(OUTPUT) $(OUTPUT)-musl

build-docker-debian:
	docker run --rm -u $(UID):$(GID) $(DOCKER_OPTS) golang:1.9 $(BUILD_CMD)
	mv $(OUTPUT) $(OUTPUT)-debian

build-docker-windows:
	docker run --rm -u $(UID):$(GID) $(DOCKER_OPTS) -e GOOS=windows golang:1.9 $(BUILD_CMD)
	mv $(OUTPUT) $(OUTPUT).exe

release: build-docker-alpine build-docker-alpine build-docker-windows

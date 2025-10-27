NAME = libzstd
COMMIT = $(shell git rev-parse --short HEAD)
GOHOSTOS = $(shell go env GOHOSTOS)
GOHOSTARCH = $(shell go env GOHOSTARCH)
PREFIX ?= $(shell go env GOPATH)

.PHONY: build

build:
	shell/build.sh



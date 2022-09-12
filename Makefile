
.PHONY: help
.ONESHELL:

.DEFAULT_GOAL := build

build:
	go build -v -o tftp-pxe-server cmd/server/main.go 
	go build -v -o tftp-pxe-client cmd/client/main.go

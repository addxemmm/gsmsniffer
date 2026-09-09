.PHONY: test vet build image export
VERSION := $(shell cat VERSION)
test:
	go test -race ./...
vet:
	go vet ./...
build:
	go build -trimpath -ldflags "-X main.version=$(VERSION)" -o bin/gsmsniffer ./cmd/server
image:
	docker build --target runtime --build-arg VERSION=$(VERSION) -f deploy/docker/Dockerfile -t gsmsniffer:$(VERSION) .
export:
	python scripts/release_export.py --output release-export

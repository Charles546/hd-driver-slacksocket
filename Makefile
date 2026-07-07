.PHONY: help build test clean docker

help:
	@echo "Available targets:"
	@echo "  build      - Build the driver binary"
	@echo "  test       - Run tests"
	@echo "  clean      - Remove built artifacts"
	@echo "  docker     - Build Docker image"

build:
	go build -o hd-driver-slacksocket ./cmd/hd-driver-slacksocket

test:
	go test -race -v ./...

clean:
	rm -f hd-driver-slacksocket

docker:
	docker build -t hd-driver-slacksocket:latest .

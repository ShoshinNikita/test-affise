default: build run

build:
	@ echo "Build..."
	@ go build -o bin/app main.go

run:
	@ echo "Run..."
	@ ./bin/app

test:
	@ echo "Run tests..."
	@ go test ./...

lint:
	@ echo "Run golangci-lint..."
	@ docker run --rm -it --network=none \
		-v $(shell go env GOCACHE):/cache/go \
		-e GOCACHE=/cache/go \
		-e GOLANGCI_LINT_CACHE=/cache/go \
		-v $(shell go env GOPATH)/pkg:/go/pkg \
		-v $(shell pwd):/app \
		-w /app \
		golangci/golangci-lint:v1.41-alpine golangci-lint run --config .golangci.yml

check: build lint test

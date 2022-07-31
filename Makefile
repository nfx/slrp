default: build

build-ui:
	npm --prefix ui run build

dev-ui:
	npm --prefix ui start

snapshot: build-ui
	goreleaser build --snapshot --rm-dist --single-target

build: build-ui
	go mod vendor
	go build -ldflags "-s -w" main.go

quick:
	go build

fmt:
	go fmt ./...

race:
	GORACE="halt_on_error=1" go run -race main.go

vendor:
	go mod vendor

test:
	go test ./... -coverprofile=coverage.txt -timeout=1m

coverage: test
	go tool cover -html=coverage.txt

.PHONY: build fmt coverage test vendor

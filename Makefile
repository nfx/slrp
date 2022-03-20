default: build

build-ui:
	npm --prefix ui run build

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
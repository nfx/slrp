default: build

clean:
	rm -fr dist ui/build ui/node_modules slrp main
	cp ui/src/favicon.ico ui/build/favicon.ico

build-ui:
	npm --prefix ui run build

dev-ui:
	npm --prefix ui start

fmt-ui:
	npm --prefix ui run prettier

snapshot: build-ui
	goreleaser build --snapshot --rm-dist --single-target

build: build-ui
	go mod vendor
	go build -ldflags "-s -w" main.go

build-go-for-docker:
	go mod vendor
	CGO_ENABLED=0 GOOS=linux go build -a -installsuffix cgo -ldflags "-s -w" -o main main.go

docker:
	docker build -t slrp:latest .

quick:
	go build

fmt:
	go fmt ./...

profiled: quick
	SLRP_PPROF_ENABLE=true slrp

heap-profile:
	go tool pprof http://localhost:6060/debug/pprof/heap

cpu-profile:
	go tool pprof http://localhost:6060/debug/pprof/profile?seconds=60

block-profile:
	go tool pprof http://localhost:6060/debug/pprof/block

pprof:
	go tool pprof -http=:8080 slrp http://127.0.0.1:6060/debug/pprof/profile

race:
	GORACE="halt_on_error=1" go run -race main.go

vendor:
	go mod vendor

test:
	go test ./... -coverprofile=coverage.txt -timeout=30s

coverage: test
	go tool cover -html=coverage.txt

.PHONY: build fmt coverage test vendor

.PHONY: build test run fmt vet
build:
	go build -o ./bin/nexora ./cmd/nexora
test:
	go test ./...
run:
	go run ./cmd/nexora
fmt:
	go fmt ./...
vet:
	go vet ./...


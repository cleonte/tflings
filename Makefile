.PHONY: build test vet
build:
	go build -o bin/tflings ./cmd/tflings
test:
	go test ./...
vet:
	go vet ./...

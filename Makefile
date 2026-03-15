.PHONY: generate test

generate:
	@echo "Generating mocks..."
	go run github.com/vektra/mockery/v2@latest

test: generate
	@echo "Running tests..."
	go test -v ./...
.PHONY: generate test

generate:
	@echo "Generating mocks..."
	mockery
	@echo "Ensuring mocks directory is ignored..."
	@grep -qxF 'mocks/' .gitignore || echo 'mocks/' >> .gitignore

test: generate
	@echo "Running tests..."
	go test -v ./...
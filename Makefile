.PHONY: run-server-locally generate test build-db deploy-db delete-db build-app deploy-app delete-app

run-server-locally:
	go run .\cmd\api\main.go

generate:
	@echo "Generating mocks..."
	go run github.com/vektra/mockery/v2@latest

test: generate
	@echo "Running tests..."
	go test -v ./...

# ---------------------------------------------------------
# DATABASE STACK (Stateful)
# ---------------------------------------------------------

build-db:
	@echo "Validating Database Infrastructure..."
	sam validate --template-file template-db.yaml

deploy-db: build-db
	@echo "Deploying Database Stack..."
	sam deploy \
		--template-file template-db.yaml \
		--stack-name reactive-bridge-db \
		--resolve-s3 \
		--capabilities CAPABILITY_IAM \
		--no-confirm-changeset

delete-db:
	@echo "Deleting Database Stack..."
	sam delete \
		--stack-name reactive-bridge-db \
		--no-prompts

# ---------------------------------------------------------
# COMPUTE STACK (Stateless)
# ---------------------------------------------------------

build-app: test
	@echo "Building App Infrastructure and Go Binaries..."
	sam build --template-file template-app.yaml

deploy-app: build-app
	@echo "Deploying App Stack..."
	sam deploy \
		--stack-name reactive-bridge-app \
		--resolve-s3 \
		--capabilities CAPABILITY_IAM \
		--no-confirm-changeset

delete-app:
	@echo "Deleting App Stack..."
	sam delete \
		--stack-name reactive-bridge-app \
		--no-prompts
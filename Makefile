.PHONY: help
help: ## Show this help message
	@echo 'Usage: make [target]'
	@echo ''
	@echo 'Targets:'
	@awk 'BEGIN {FS = ":.*?## "} /^[a-zA-Z_-]+:.*?## / {printf "  %-20s %s\n", $$1, $$2}' $(MAKEFILE_LIST)

.PHONY: deps
deps: ## Install dependencies
	go mod download
	go mod tidy

.PHONY: build
build: ## Build the project
	go build -v ./...

.PHONY: test
test: ## Run unit tests
	go test -v -race -cover ./pkg/... ./internal/...

.PHONY: test-short
test-short: ## Run unit tests in short mode
	go test -v -short ./pkg/... ./internal/...

.PHONY: test-integration
test-integration: ## Run integration tests (requires AWS credentials)
	@echo "Running integration tests..."
	@echo "Make sure you have AWS credentials configured"
	cd test/integration && go test -v -timeout 30m .

.PHONY: test-integration-cleanup
test-integration-cleanup: ## Run integration tests and force cleanup
	cd test/integration && go test -v -timeout 30m -run TestTokenizerIntegration .

.PHONY: test-all
test-all: test test-integration ## Run all tests

.PHONY: fmt
fmt: ## Format code
	go fmt ./...

.PHONY: vet
vet: ## Run go vet
	go vet ./...

.PHONY: lint
lint: ## Run golangci-lint
	@which golangci-lint > /dev/null || (echo "golangci-lint not installed. Run: go install github.com/golangci/golangci-lint/cmd/golangci-lint@latest" && exit 1)
	golangci-lint run

.PHONY: clean
clean: ## Clean build artifacts
	go clean -cache
	rm -rf dist/

.PHONY: coverage
coverage: ## Generate test coverage report
	go test -coverprofile=coverage.out ./pkg/... ./internal/...
	go tool cover -html=coverage.out -o coverage.html
	@echo "Coverage report generated: coverage.html"

.PHONY: benchmark
benchmark: ## Run benchmarks
	go test -bench=. -benchmem ./pkg/...

.PHONY: docker-build
docker-build: ## Build Docker image
	docker build -t tegridy-tokens:latest .

.PHONY: example
example: ## Run the example
	go run cmd/example/main.go

.PHONY: terraform-init
terraform-init: ## Initialize Terraform for tests
	cd test/terraform && terraform init

.PHONY: terraform-plan
terraform-plan: terraform-init ## Plan Terraform changes
	cd test/terraform && terraform plan

.PHONY: terraform-apply
terraform-apply: terraform-init ## Apply Terraform changes
	cd test/terraform && terraform apply -auto-approve

.PHONY: terraform-destroy
terraform-destroy: ## Destroy Terraform resources
	cd test/terraform && terraform destroy -auto-approve

.PHONY: ci
ci: deps vet test ## Run CI checks

.PHONY: install-tools
install-tools: ## Install development tools
	go install github.com/golangci/golangci-lint/cmd/golangci-lint@latest
	go install github.com/gruntwork-io/terratest/cmd/terratest_log_parser@latest
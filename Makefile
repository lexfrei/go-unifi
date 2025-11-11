.PHONY: generate
generate:
	@echo "Generating client code from OpenAPI specification..."
	@go run github.com/oapi-codegen/oapi-codegen/v2/cmd/oapi-codegen --config .oapi-codegen.yaml openapi.yaml

.PHONY: lint
lint:
	@echo "Running golangci-lint..."
	@golangci-lint run

.PHONY: test
test:
	@echo "Running tests..."
	@go test -v -race -coverprofile=coverage.out ./...

.PHONY: tidy
tidy:
	@echo "Tidying go.mod..."
	@go mod tidy

.PHONY: all
all: generate tidy lint test

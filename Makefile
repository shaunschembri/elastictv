LINTER_VERSION := v1.61.0

lint:
	@ go install github.com/golangci/golangci-lint/cmd/golangci-lint@$(LINTER_VERSION)
	@ ~/go/bin/golangci-lint run --verbose --fix --timeout 30s

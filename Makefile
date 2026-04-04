.PHONY: setup

setup: ## Install all dev tools (requires Go)
	@command -v task >/dev/null 2>&1 || go install github.com/go-task/task/v3/cmd/task@latest
	@command -v goose >/dev/null 2>&1 || go install github.com/pressly/goose/v3/cmd/goose@latest
	@command -v golangci-lint >/dev/null 2>&1 || go install github.com/golangci/golangci-lint/v2/cmd/golangci-lint@latest
	@command -v goreman >/dev/null 2>&1 || go install github.com/mattn/goreman@latest
	@command -v air >/dev/null 2>&1 || go install github.com/air-verse/air@latest
	@command -v goimports >/dev/null 2>&1 || go install golang.org/x/tools/cmd/goimports@latest
	@echo "All tools installed. Use 'task --list' to see available commands."

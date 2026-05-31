GO_DIR := go

.DEFAULT_GOAL := check

.PHONY: check analyze setup

check:
	@echo "==> Cleaning build and test cache..."
	@cd $(GO_DIR) && go clean -cache -testcache
	@echo "==> Building..."
	@cd $(GO_DIR) && go build ./...
	@echo "==> Vetting..."
	@cd $(GO_DIR) && go vet ./...
	@echo "==> Testing..."
	@cd $(GO_DIR) && go test ./...
	@echo "==> All checks passed."

analyze:
	@echo "==> Running staticcheck..."
	@cd $(GO_DIR) && staticcheck ./...
	@echo "==> Running golangci-lint..."
	@cd $(GO_DIR) && golangci-lint run
	@echo "==> Checking go.mod is tidy..."
	@cd $(GO_DIR) && go mod tidy && git diff --exit-code go.mod go.sum
	@echo "==> Analysis complete."

setup:
	@echo "==> Installing staticcheck..."
	@go install honnef.co/go/tools/cmd/staticcheck@latest
	@echo "==> Installing golangci-lint..."
	@go install github.com/golangci/golangci-lint/cmd/golangci-lint@latest
	@echo "==> Done. Ensure $$(go env GOPATH)/bin is on your PATH."

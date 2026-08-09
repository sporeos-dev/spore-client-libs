GO_DIR     := go
PARSER_DIR := parser
CLIENT_DIR := spore_client

.DEFAULT_GOAL := check

.PHONY: check analyze setup release debug clean test cross

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

release:
	@echo "==> Building Go..."
	@cd $(GO_DIR) && go build ./...
	@echo "==> Building parser (native)..."
	@$(MAKE) -C $(PARSER_DIR) -f Makefile.macos-arm release
	@echo "==> Building spore_client (native)..."
	@$(MAKE) -C $(CLIENT_DIR) -f Makefile.macos-arm release
	@echo "==> Running tests..."
	@$(MAKE) -C $(PARSER_DIR) test
	@$(MAKE) -C $(CLIENT_DIR) test
	@echo "==> Done."

cross:
	@echo "==> Building parser (all platforms)..."
	@$(MAKE) -C $(PARSER_DIR) release
	@echo "==> Building spore_client (all platforms)..."
	@$(MAKE) -C $(CLIENT_DIR) release

debug:
	@echo "==> Building parser (native, debug)..."
	@$(MAKE) -C $(PARSER_DIR) -f Makefile.macos-arm debug
	@echo "==> Building spore_client (native, debug)..."
	@$(MAKE) -C $(CLIENT_DIR) -f Makefile.macos-arm debug

clean:
	@echo "==> Cleaning parser builds..."
	@$(MAKE) -C $(PARSER_DIR) clean
	@echo "==> Cleaning spore_client builds..."
	@$(MAKE) -C $(CLIENT_DIR) clean

test:
	@echo "==> Running parser tests..."
	@$(MAKE) -C $(PARSER_DIR) test
	@echo "==> Running spore_client tests..."
	@$(MAKE) -C $(CLIENT_DIR) test

analyze:
	@echo "==> Running tests with race detector..."
	@cd $(GO_DIR) && go test -race ./...

	@echo "==> Generating coverage report..."
	@cd $(GO_DIR) && go test -coverprofile=coverage.out ./...
	@cd $(GO_DIR) && go tool cover -func=coverage.out

	@echo "==> Running staticcheck..."
	@cd $(GO_DIR) && staticcheck ./...

	@echo "==> Running golangci-lint..."
	@cd $(GO_DIR) && golangci-lint run

	@echo "==> Checking go.mod is tidy..."
	@cd $(GO_DIR) && go mod tidy && git diff --exit-code go.mod go.sum

	@echo "==> Running govulncheck..."
	@cd $(GO_DIR) && govulncheck ./...

	@echo "==> Analysis complete."

setup:
	@echo "==> Installing staticcheck..."
	@go install honnef.co/go/tools/cmd/staticcheck@latest
	@echo "==> Installing golangci-lint..."
	@go install github.com/golangci/golangci-lint/cmd/golangci-lint@latest
	@echo "==> Installing govulncheck..."
	@go install golang.org/x/vuln/cmd/govulncheck@latest
	@echo "==> Done. Ensure $$(go env GOPATH)/bin is on your PATH."

GO_REF_DIR := go_ref
GO_DIR     := spore_go
PARSER_DIR := parser
CLIENT_DIR := spore_c

.DEFAULT_GOAL := check

.PHONY: check analyze setup release debug clean test cross

check:
	@echo "==> Cleaning go_ref build and test cache..."
	@cd $(GO_REF_DIR) && go clean -cache -testcache
	@echo "==> Building go_ref..."
	@cd $(GO_REF_DIR) && go build ./...
	@echo "==> Vetting go_ref..."
	@cd $(GO_REF_DIR) && go vet ./...
	@echo "==> Testing go_ref..."
	@cd $(GO_REF_DIR) && go test ./...
	@echo "==> All checks passed."

release:
	@echo "==> Building parser (native)..."
	@$(MAKE) -C $(PARSER_DIR) -f Makefile.macos-arm release
	@echo "==> Building spore_c (native)..."
	@$(MAKE) -C $(CLIENT_DIR) -f Makefile.macos-arm release
	@echo "==> Building spore_go..."
	@cd $(GO_DIR) && go build ./...
	@echo "==> Building go_ref..."
	@cd $(GO_REF_DIR) && go build ./...
	@echo "==> Running C tests..."
	@$(MAKE) -C $(PARSER_DIR) test
	@$(MAKE) -C $(CLIENT_DIR) test
	@echo "==> Running go_ref tests..."
	@cd $(GO_REF_DIR) && go test ./...
	@echo "==> Done."

cross:
	@echo "==> Building parser (all platforms)..."
	@$(MAKE) -C $(PARSER_DIR) release
	@echo "==> Building spore_c (all platforms)..."
	@$(MAKE) -C $(CLIENT_DIR) release

debug:
	@echo "==> Building parser (native, debug)..."
	@$(MAKE) -C $(PARSER_DIR) -f Makefile.macos-arm debug
	@echo "==> Building spore_c (native, debug)..."
	@$(MAKE) -C $(CLIENT_DIR) -f Makefile.macos-arm debug

clean:
	@echo "==> Cleaning parser builds..."
	@$(MAKE) -C $(PARSER_DIR) clean
	@echo "==> Cleaning spore_c builds..."
	@$(MAKE) -C $(CLIENT_DIR) clean

test:
	@echo "==> Running parser tests..."
	@$(MAKE) -C $(PARSER_DIR) test
	@echo "==> Running spore_c tests..."
	@$(MAKE) -C $(CLIENT_DIR) test
	@echo "==> Running go_ref tests..."
	@cd $(GO_REF_DIR) && go test ./...

analyze:
	@echo "==> Running go_ref tests with race detector..."
	@cd $(GO_REF_DIR) && go test -race ./...

	@echo "==> Generating go_ref coverage report..."
	@cd $(GO_REF_DIR) && go test -coverprofile=coverage.out ./...
	@cd $(GO_REF_DIR) && go tool cover -func=coverage.out

	@echo "==> Running staticcheck..."
	@cd $(GO_REF_DIR) && staticcheck ./...

	@echo "==> Running golangci-lint..."
	@cd $(GO_REF_DIR) && golangci-lint run

	@echo "==> Checking go_ref go.mod is tidy..."
	@cd $(GO_REF_DIR) && go mod tidy && git diff --exit-code go.mod go.sum

	@echo "==> Running govulncheck..."
	@cd $(GO_REF_DIR) && govulncheck ./...

	@echo "==> Analysis complete."

setup:
	@echo "==> Installing staticcheck..."
	@go install honnef.co/go/tools/cmd/staticcheck@latest
	@echo "==> Installing golangci-lint..."
	@go install github.com/golangci/golangci-lint/cmd/golangci-lint@latest
	@echo "==> Installing govulncheck..."
	@go install golang.org/x/vuln/cmd/govulncheck@latest
	@echo "==> Done. Ensure $$(go env GOPATH)/bin is on your PATH."

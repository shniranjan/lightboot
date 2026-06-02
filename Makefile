.PHONY: build clean web docs docker docker-up docker-down docker-logs run help

# Binary output directory
BIN_DIR := bin
BINARY := $(BIN_DIR)/lightboot

# Go build flags
GO := go
LDFLAGS := -s -w

build: ## Compile LightBoot for current OS/arch
	@mkdir -p $(BIN_DIR)
	$(GO) build -ldflags="$(LDFLAGS)" -o $(BINARY) ./cmd/lightboot/
	@echo "Built: $(BINARY)"

clean: ## Remove build artifacts
	rm -rf $(BIN_DIR)
	rm -rf web/dist

web: ## Build Vue.js frontend
	@echo "Building Vue.js frontend..."
	cd web && npm install && npx vite build
	@echo "Frontend built to web/dist/"



docs: ## Build MkDocs documentation site
	@echo "Building MkDocs documentation..."
	.mkdocs-venv/bin/mkdocs build
	@echo "Documentation built to site/"

docker: ## Build Docker image
	docker build -t lightboot:latest -f docker/Dockerfile .

docker-up: ## Build and start with docker-compose
	docker compose up -d --build

docker-down: ## Stop docker-compose services
	docker compose down

docker-logs: ## Follow docker-compose logs
	docker compose logs -f

run: build ## Build and run LightBoot
	./$(BINARY)

help: ## Show this help
	@grep -E '^[a-zA-Z_-]+:.*?## .*$$' $(MAKEFILE_LIST) | sort | awk 'BEGIN {FS = ":.*?## "}; {printf "\033[36m%-15s\033[0m %s\n", $$1, $$2}'

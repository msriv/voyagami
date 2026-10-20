# Voyagami monorepo commands. See CLAUDE.md.
#
# Go targets only run once services/api has a go.mod (VOY-91), so the
# scaffold stays green while the API does not exist yet.

API_DIR := services/api
HAS_API := $(wildcard $(API_DIR)/go.mod)

.PHONY: setup dev lint test typecheck format api-client

setup: ## Install toolchain dependencies
	corepack enable
	pnpm install --frozen-lockfile
ifneq ($(HAS_API),)
	cd $(API_DIR) && go mod download
endif

dev: ## Run the API (the app, VOY-98, adds itself here)
ifneq ($(HAS_API),)
	cd $(API_DIR) && go run ./cmd/api
else
	@echo "make dev: nothing to run yet. The API (VOY-91) and app (VOY-98) add themselves here."
endif

lint: ## Format check, ESLint, typecheck and Go lint
	pnpm run lint
	pnpm run typecheck
ifneq ($(HAS_API),)
	cd $(API_DIR) && test -z "$$(gofmt -l .)" || (gofmt -l . && echo "gofmt: files above need formatting" && exit 1)
	cd $(API_DIR) && golangci-lint run --config ../../.golangci.yml ./...
endif

test: ## Run all tests
	pnpm run test
ifneq ($(HAS_API),)
	cd $(API_DIR) && go test ./...
endif

format: ## Rewrite files with Prettier and gofmt
	pnpm run format
ifneq ($(HAS_API),)
	cd $(API_DIR) && gofmt -w .
endif

api-client: ## Generate the typed API client from OpenAPI (VOY-16)
	@echo "make api-client: no OpenAPI spec yet. VOY-92 adds the spec and VOY-16 the generator."

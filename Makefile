# Voyagami monorepo commands. See CLAUDE.md.
#
# Go targets only run once services/api has a go.mod (VOY-91), so the
# scaffold stays green while the API does not exist yet.

API_DIR := services/api
HAS_API := $(wildcard $(API_DIR)/go.mod)

.PHONY: setup dev lint test typecheck format api-client deploy-dev

setup: ## Install toolchain dependencies
	corepack enable
	pnpm install --frozen-lockfile
ifneq ($(HAS_API),)
	cd $(API_DIR) && go mod download
endif

dev: ## Run the API (start the app with: pnpm --filter @voyagami/mobile start)
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

test: ## Run all tests (starts the Firestore emulator for the API when Java is installed)
	pnpm run test
ifneq ($(HAS_API),)
	@if command -v java >/dev/null 2>&1; then \
		pnpm exec firebase emulators:exec --only firestore --project demo-voyagami \
			"cd $(API_DIR) && REQUIRE_EMULATOR=1 go test ./..."; \
	else \
		echo "make test: Java not found, so the Firestore emulator tests are skipped. Install Java 21 to run them."; \
		cd $(API_DIR) && go test ./...; \
	fi
endif

format: ## Rewrite files with Prettier and gofmt
	pnpm run format
ifneq ($(HAS_API),)
	cd $(API_DIR) && gofmt -w .
endif

api-client: ## Generate code from api/openapi.yaml (Go server now; TypeScript client is VOY-16)
ifneq ($(HAS_API),)
	cd $(API_DIR) && go generate ./...
endif
	@echo "make api-client: the TypeScript client arrives with VOY-16."

deploy-dev: ## Build the API image and deploy it to the dev Cloud Run service (needs docker and gcloud)
	infra/scripts/deploy-api-dev.sh

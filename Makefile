export

LOCAL_BIN:=$(CURDIR)/bin
PATH:=$(LOCAL_BIN):$(PATH)
DB_PATH:=$(CURDIR)/database/postgres

help: ## Display this help screen
	@awk 'BEGIN {FS = ":.*##"; printf "\nUsage:\n  make \033[36m<target>\033[0m\n"} /^[a-zA-Z_.-]+:.*?##/ { printf "  \033[36m%-15s\033[0m %s\n", $$1, $$2 } /^##@/ { printf "\n\033[1m%s\033[0m\n", substr($$0, 5) } ' $(MAKEFILE_LIST)
.PHONY: help

deps: ### deps tidy + verify
	go mod tidy && go mod verify
.PHONY: deps

deps.bin: ### install tools (mandatory for development)
	GOBIN=$(LOCAL_BIN) go install -tags 'postgres' github.com/golang-migrate/migrate/v4/cmd/migrate@latest
	GOBIN=$(LOCAL_BIN) go install github.com/golangci/golangci-lint/cmd/golangci-lint@latest
	GOBIN=$(LOCAL_BIN) go install golang.org/x/vuln/cmd/govulncheck@latest
.PHONY: deps.bin

deps.audit: ### check dependencies vulnerabilities
	$(LOCAL_BIN)/govulncheck ./...
.PHONY: deps.audit

generate.domains: ### domains=$DOMAIN (generate domains including sqlc.yaml)
		@if [ -z "$(domains)" ]; then \
			echo "Please set the domains variable"; \
			echo "Example: make generate.domains domains=users"; \
			exit 1; \
		fi
		mkdir -p ./internal/domains/$(domains)/service \
			./internal/domains/$(domains)/handler \
			./internal/domains/$(domains)/repository \
			$(DB_PATH)/domains/$(domains)
		touch $(DB_PATH)/domains/$(domains)/schema.sql $(DB_PATH)/domains/$(domains)/queries.sql
		@echo "version: \"2\"" > $(DB_PATH)/domains/$(domains)/sqlc.yaml
		@echo "sql:" >> $(DB_PATH)/domains/$(domains)/sqlc.yaml
		@echo "  - name: \"$(domains)\"" >> $(DB_PATH)/domains/$(domains)/sqlc.yaml
		@echo "    engine: \"postgresql\"" >> $(DB_PATH)/domains/$(domains)/sqlc.yaml
		@echo "    schema: \"./schema.sql\"" >> $(DB_PATH)/domains/$(domains)/sqlc.yaml
		@echo "    queries: \"./queries.sql\"" >> $(DB_PATH)/domains/$(domains)/sqlc.yaml
		@echo "    gen:" >> $(DB_PATH)/domains/$(domains)/sqlc.yaml
		@echo "      go:" >> $(DB_PATH)/domains/$(domains)/sqlc.yaml
		@echo "        package: \"sqlc\"" >> $(DB_PATH)/domains/$(domains)/sqlc.yaml
		@echo "        sql_package: \"pgx/v5\"" >> $(DB_PATH)/domains/$(domains)/sqlc.yaml
		@echo "        out: \"./sqlc\"" >> $(DB_PATH)/domains/$(domains)/sqlc.yaml
		@echo "        emit_json_tags: true" >> $(DB_PATH)/domains/$(domains)/sqlc.yaml
		@echo "        emit_db_tags: true" >> $(DB_PATH)/domains/$(domains)/sqlc.yaml
		@echo "        emit_methods_with_db_argument: true" >> $(DB_PATH)/domains/$(domains)/sqlc.yaml
		@echo "        emit_interface: true" >> $(DB_PATH)/domains/$(domains)/sqlc.yaml
		@echo "Domain structure created at ./internal/domains/$(domains) and sqlc.yaml at $(DB_PATH)/domains/$(domains)"
		@echo "Please edit the schema.sql and queries.sql files to add your own queries"
.PHONY: generate.domains

generate.sqlc: ### generate sqlc code
	@for domain in $$(find $(DB_PATH)/domains -mindepth 1 -maxdepth 1 -type d -exec basename {} \;); do \
		if [ -f "$$(find $(DB_PATH)/domains/$$domain -name sqlc.yaml)" ]; then \
			echo "Generating sqlc for domain $$domain"; \
			go run github.com/sqlc-dev/sqlc/cmd/sqlc generate -f "$$(find $(DB_PATH)/domains/$$domain -name sqlc.yaml)"; \
		else \
			echo "No sqlc.yaml found for domain $$domain"; \
		fi; \
	done
.PHONY: generate.sqlc

generate.swag: ### generate swagger docs
	go run github.com/swaggo/swag/cmd/swag init -g ./internal/delivery/http/router.go
.PHONY: generate.swag

generate.mock: ### generate mock
	@for domain in $$(find ./internal/domains -mindepth 1 -maxdepth 1 -type d -exec basename {} \;); do \
		mkdir -p ./internal/domains/$$domain/mock; \
		for dir in repository service; do \
			if [ -d "./internal/domains/$$domain/$$dir" ]; then \
				f=$$(find "./internal/domains/$$domain/$$dir" -name "*.go" -not -path "*/mock/*" -type f | xargs grep -l "type.*interface\|type.*Interface" 2>/dev/null || true); \
				if [ -n "$$f" ]; then \
					echo "$$f" | while read file; do \
						if [ -n "$$file" ]; then \
							dest_file="./internal/domains/$$domain/mock/$$(basename $${file%.*})_mock.go"; \
							go run go.uber.org/mock/mockgen -source="$$file" -destination="$$dest_file" -package=mock || echo "    ERROR: Failed to generate mock for $$file"; \
						fi \
					done; \
				fi; \
			fi; \
		done; \
	done
	go generate -run="mockgen" ./pkg/...
	@echo "Mock generation completed"
.PHONY: generate.mock

generate: generate.sqlc ### generate code
	go generate ./...
.PHONY: generate

migrate.up: ### run database migrations up
	@if [ -z "$(dsn)" ]; then \
  		echo "Please set the dsn variable"; \
		echo "Example: make migrate.up dsn=postgres://postgres:postgres@localhost:5432/postgres?sslmode=disable"; \
		exit 1; \
	fi
	@echo "Running database migrations up"
	@$(LOCAL_BIN)/migrate -path $(DB_PATH)/migrations -database "$(dsn)" up

migrate.down: ### run database migrations down
	@if [ -z "$(dsn)" ]; then \
  		echo "Please set the dsn variable"; \
		echo "Example: make migrate.down dsn=postgres://postgres:postgres@localhost:5432/postgres?sslmode=disable"; \
		exit 1; \
	fi
	@echo "Running database migrations down"
	@$(LOCAL_BIN)/migrate -path $(DB_PATH)/migrations -database "$(dsn)" down

migrate.step-up: ### run database migrations step up
	@if [ -z "$(dsn)" ]; then \
  		echo "Please set the dsn variable"; \
		echo "Example: make migrate.step-up dsn=postgres://postgres:postgres@localhost:5432/postgres?sslmode=disable"; \
		exit 1; \
	fi
	@echo "Running database migrations step up"
	@$(LOCAL_BIN)/migrate -path $(DB_PATH)/migrations -database "$(dsn)" step-up

migrate.step-down: ### run database migrations step down
	@if [ -z "$(dsn)" ]; then \
  		echo "Please set the dsn variable"; \
		echo "Example: make migrate.step-down dsn=postgres://postgres:postgres@localhost:5432/postgres?sslmode=disable"; \
		exit 1; \
	fi
	@echo "Running database migrations step down"
	@$(LOCAL_BIN)/migrate -path $(DB_PATH)/migrations -database "$(dsn)" step-down

migrate.create: ### create migration
	@if [ -z "$(name)" ]; then \
  		echo "Please set the name variable"; \
		echo "Example: make migrate.create name=create_users_table"; \
		exit 1; \
	fi
	@echo "Creating migration $(name)"
	@$(LOCAL_BIN)/migrate create -ext sql -dir $(DB_PATH)/migrations -seq $(name)

migrate.force: ### force migration
	@if [ -z "$(version)" ]; then \
  		echo "Please set the version variable"; \
		echo "Example: make migrate.force version=1"; \
		exit 1; \
	fi
	@echo "Forcing migration to version $(version)"
	@$(LOCAL_BIN)/migrate force -path $(DB_PATH)/migrations -version $(version)

lint: ### check by golangci linter
	$(LOCAL_BIN)/golangci-lint run
.PHONY: linter-golangci

test: generate generate.mock ### run test
	@if ! -d ./tmp ]; then \
		mkdir -p ./tmp; \
	fi
	go test -v -race -covermode atomic -coverprofile=tmp/coverage.txt ./internal/...
.PHONY: test

coverage: ### show coverage
	go tool cover -html=tmp/coverage.txt

dev: generate ### Run dev
	go run github.com/air-verse/air -c ./.air.toml
.PHONY: dev

clean: ### clean
	@rm -rf ./bin ./tmp ./docs
	@for domain in $$(find ./internal/domains -mindepth 1 -maxdepth 1 -type d -exec basename {} \;); do \
		rm -rf ./internal/domains/$$domain/mock; \
	done
	@for pkg in $$(find ./pkg -mindepth 1 -maxdepth 1 -type d -exec basename {} \;); do \
		rm -rf ./pkg/$$pkg/mock; \
	done
.PHONY: clean

ENV_FILE=./.env
MIGRATIONS_DIR=migrations

run:
	nodemon

env:
	@set -a; [ -f $(ENV_FILE) ] && . $(ENV_FILE); set +a; echo "env loaded"

migrate-up:
	@set -a; [ -f $(ENV_FILE) ] && . $(ENV_FILE); set +a; \
	migrate -path ./migrations -database "$$DB_DSN" up

migrate-down-1:
	@set -a; [ -f $(ENV_FILE) ] && . $(ENV_FILE); set +a; \
	migrate -path ./migrations -database "$$DB_DSN" down 1

migrate-force:
	@set -a; [ -f $(ENV_FILE) ] && . $(ENV_FILE); set +a; \
	migrate -path ./migrations -database "$$DB_DSN" force $(v)

migrate-version:
	@set -a; [ -f $(ENV_FILE) ] && . $(ENV_FILE); set +a; \
	migrate -path ./migrations -database "$$DB_DSN" version

# Usage: make migration name=add_posts
migration:
	@if [ -z "$(name)" ]; then \
		echo "❌ name is required. Example: make migration name=add_posts"; \
		exit 1; \
	fi
	@mkdir -p $(MIGRATIONS_DIR)
	@ts=$$(date +%Y%m%d%H%M%S); \
	base="$(MIGRATIONS_DIR)/$${ts}_$(name)"; \
	touch "$${base}.up.sql" "$${base}.down.sql"; \
	echo "-- +migrate Up" > "$${base}.up.sql"; \
	echo "-- +migrate Down" > "$${base}.down.sql"; \
	echo "✅ created:"; \
	echo "  $${base}.up.sql"; \
	echo "  $${base}.down.sql"

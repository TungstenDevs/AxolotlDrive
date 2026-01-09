# =========================
# Config
# =========================

MIGRATIONS_DIR := migrations
MIGRATE := migrate
ENV_FILE ?= .env.development

# Load env file if exists (simple KEY=VALUE, no export, no quotes)
ifneq ("$(wildcard $(ENV_FILE))","")
    include $(ENV_FILE)
    export
endif

ifndef DB_URL
$(error DB_URL is not set. Make sure $(ENV_FILE) has DB_URL)
endif

# =========================
# App
# =========================

.PHONY: build
build:
	go build -o bin/axolotldrive cmd/main.go


.PHONY: run
run:
	make build
	./bin/axolotldrive

.PHONY: test
test:
	go test ./...

# =========================
# Migrations
# =========================

.PHONY: migrate-up
migrate-up:
	$(MIGRATE) -path $(MIGRATIONS_DIR) -database "$(DB_URL)" up

.PHONY: migrate-down
migrate-down:
	$(MIGRATE) -path $(MIGRATIONS_DIR) -database "$(DB_URL)" down 1

.PHONY: migrate-version
migrate-version:
	$(MIGRATE) -path $(MIGRATIONS_DIR) -database "$(DB_URL)" version

.PHONY: migrate-force
migrate-force:
	$(MIGRATE) -path $(MIGRATIONS_DIR) -database "$(DB_URL)" force $(version)

.PHONY: migrate-create
migrate-create:
	$(MIGRATE) create -ext sql -dir $(MIGRATIONS_DIR) $(name)

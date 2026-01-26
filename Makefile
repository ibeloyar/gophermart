GO = go
MAKE = make
DB_HOST=192.168.0.104
DB_USER=gophermart
DB_NAME=gophermart
DB_PASS=gophermart
DB_PORT=5432
DB_STRING="postgres://$(DB_NAME):$(DB_PASS)@$(DB_HOST):$(DB_PORT)/$(DB_NAME)?sslmode=disable"
DB_MIGRATIONS_PATH="./migrations"

.DEFAULT_GOAL := help

.PHONY: build
build:
	$(GO) build -o cmd/gophermart/gophermart cmd/gophermart/main.go

.PHONY: run
run:
	$(GO) run cmd/gophermart/main.go -d $(DB_STRING)

.PHONY: run_accrual_linux
run_accrual_linux:
	./cmd/accrual/accrual_linux_amd64 -a localhost:4000

.PHONY: mock
mock:
	@echo "Generating mock for StorageRepo..."
	mockgen -destination=internal/repository/mocks/mock.go -package=-source=internal/service/service.go StorageRepo
	@echo "Generating mock for service.Service..."
	mockgen -destination=internal/service/mocks/service_mock.go -package=service -source=internal/controller/http/handlers.go

.PHONY: test
test:
	$(GO) test -v ./... | { grep -v 'no test files'; true; }

.PHONY: test_cover
test_cover:
	$(GO) test -coverprofile=coverage.out ./...
	cat coverage.out | grep -v '/mocks\|/test\|/vendor\|/internal/model' > coverage.filtered.out
	$(GO) tool cover -func=coverage.filtered.out
	rm coverage.out coverage.filtered.out

.PHONY: test_main
test_main:
	$(MAKE) build
	gophermarttest \
	-test.v \
	-gophermart-binary-path=cmd/gophermart/gophermart \
	-gophermart-host=localhost \
	-gophermart-port=8080 \
	-gophermart-database-uri="$(DB_STRING)" \
	-accrual-binary-path=cmd/accrual/accrual_linux_amd64 \
	-accrual-host=localhost \
	-accrual-port=4000 \
	-accrual-database-uri="postgresql://postgres:postgres@postgres/praktikum?sslmode=disable"

.PHONY: migrate-up
migrate-up:
	migrate \
	-path $(DB_MIGRATIONS_PATH) \
	-database $(DB_STRING) up

.PHONY: migrate-down
migrate-down:
	migrate \
	-path $(DB_MIGRATIONS_PATH) \
	-database $(DB_STRING) down

.PHONY: migrate-create
migrate-create:
ifdef NAME
	migrate create \
    	-ext sql \
    	-dir $(DB_MIGRATIONS_PATH) \
    	-seq $(NAME)
else
	@echo "Require variable NAME not found"
endif

.PHONY: install-tools
install-tools:
	go install -tags 'postgres' github.com/golang-migrate/migrate/v4/cmd/migrate@latest # golang-migrate CLI
	go install github.com/golang/mock/mockgen@latest  # mocks for tests

.PHONY: help
help:
	@echo "command           | description"
	@echo "===================================================="
	@echo "build             | build gophermart"
	@echo "run               | run gophermart server"
	@echo "run_accrual_linux | run accrual server for linux"
	@echo "mock              | generate repositories mocks for tests"
	@echo "test              | run tests with 'clean' out"
	@echo "test_cover        | run tests with coverage info"
	@echo "test_main         | run main integrations tests"
	@echo "migrate-up        | run UP migrations"
	@echo "migrate-down      | run DOWN migrations"
	@echo "migrate-create    | run create migration with NAME; EXAMPLE: make NAME=add_users migrate-create"
	@echo "install-tools     | install libs for work with project"

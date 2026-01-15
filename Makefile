.PHONY: test_main
test_main:
	gophermarttest \
	-test.v -test.run=^TestGophermart$ \
	-gophermart-binary-path=cmd/gophermart/gophermart \
	-gophermart-host=localhost \
	-gophermart-port=8080 \
	-gophermart-database-uri="postgresql://postgres:postgres@postgres/praktikum?sslmode=disable" \
	-accrual-binary-path=cmd/accrual/accrual_linux_amd64 \
	-accrual-host=localhost \
	-accrual-port=4000 \
	-accrual-database-uri="postgresql://postgres:postgres@postgres/praktikum?sslmode=disable"


.PHONY: help
help:
	@echo "command           | description"
	@echo "===================================================="
	@echo "test_main         | run main integrations tests"

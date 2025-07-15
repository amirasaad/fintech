test:
	go test -p 4 -v $$(go list ./... | grep -v '/internal' | grep -v '/api' | grep -v '/cmd')

test-unit:
	go test -p 4 -v -coverprofile unit.cover.out ./pkg/...

test-integration:
	go test -p 4 -v -coverprofile integration.cover.out ./infra/...

test-e2e:
	go test -p 4 -v -coverprofile e2e.cover.out ./webapi/...

cov:
	go test -p 4 -v -coverprofile cover.out $$(go list ./... | grep -v '/internal' | grep -v '/api' | grep -v '/cmd')
cov_report: cov
	go tool cover -html cover.out -o docs/coverage.html

run:
	go run cmd/server/main.go

migrate-up:
	@echo "Applying migrations..."
	@migrate -database "$(DATABASE_URL)" -path internal/migrations up

migrate-down:
	@echo "Reverting migrations..."
	@migrate -database "$(DATABASE_URL)" -path internal/migrations down ${n}

migrate-create:
	@if [ -z "$(name)" ]; then \
		echo "Usage: make migrate-create name=<migration_name>"; \
		exit 1; \
	fi
	@migrate create -ext sql -dir internal/migrations -seq $(name)

.PHONY: test cov cov_report run migrate-up migrate-down migrate-create

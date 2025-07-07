test:
	go test -p 4 -v $(go list ./... | grep -v '/internal' | grep -v '/api')
cov:
	go test -p 4 -v -coverprofile cover.out $(go list ./... | grep -v '/internal' | grep -v '/api')
cov_report: cov
	go tool cover -html cover.out -o docs/cover.html

run:
	go run cmd/server/main.go

migrate-up:
	@echo "Applying migrations..."
	@migrate -database "postgres://postgres:password@localhost:5432/fintech?sslmode=disable" -path migrations up

migrate-down:
	@echo "Reverting migrations..."
	@migrate -database "postgres://postgres:password@localhost:5432/fintech?sslmode=disable" -path migrations down

migrate-create:
	@read -p "Enter migration name: " name; \
	@migrate create -ext sql -dir migrations -seq $name

.PHONY: test cov cov_report run migrate-up migrate-down migrate-create

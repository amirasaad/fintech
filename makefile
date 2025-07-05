test:
	go test -v $$(go list ./... | grep -v '/internal' | grep -v '/api')
cov:
	go test -v -coverprofile cover.out $$(go list ./... | grep -v '/internal' | grep -v '/api')
cov_report: cov
	go tool cover -html cover.out -o docs/cover.html

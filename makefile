test:
	go test -v $$(go list ./... | grep -v '/test')
cov:
	go test -v -coverprofile cover.out $$(go list ./... | grep -v '/test')
cov_report: cov
	go tool cover -html cover.out -o cover.html

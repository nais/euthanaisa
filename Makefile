local:
	go run cmd/euthanaisa/main.go -log-level=debug -pushgateway-url=https://prometheus-pushgateway.dev.dev-nais.cloud.nais.io

linux-binary:
	GOOS=linux GOARCH=amd64 go build -o bin/euthanaisa cmd/euthanaisa/main.go

check: staticcheck vulncheck deadcode

staticcheck:
	go run honnef.co/go/tools/cmd/staticcheck@latest ./...

vulncheck:
	go run golang.org/x/vuln/cmd/govulncheck@latest ./...

deadcode:
	go run golang.org/x/tools/cmd/deadcode@latest -test ./...

fmt:
	go run mvdan.cc/gofumpt@latest -w ./

helm-lint:
	helm lint --strict ./charts

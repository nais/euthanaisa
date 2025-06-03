local:
	go run cmd/euthanaisa/main.go -log-level=debug -pushgateway-url=https://prometheus-pushgateway.dev.dev-nais.cloud.nais.io

linux-binary:
	GOOS=linux GOARCH=amd64 go build -o bin/euthanaisa cmd/euthanaisa/main.go

test:
	go test -cover ./...

test-coverage:
	go test -coverprofile=coverage.out ./...
	go tool cover -html=coverage.out

check: fmt vulncheck deadcode gosec staticcheck goimport

goimport:
	@echo "Running goimport..."
	go run golang.org/x/tools/cmd/goimports@latest -l -w .

fmt:
	@echo "Running go fmt..."
	go fmt ./...

staticcheck:
	@echo "Running staticcheck..."
	go run honnef.co/go/tools/cmd/staticcheck@latest -f=stylish ./...

vulncheck:
	@echo "Running vulncheck..."
	go run golang.org/x/vuln/cmd/govulncheck@latest ./...

deadcode:
	@echo "Running deadcode..."
	go run golang.org/x/tools/cmd/deadcode@latest ./...

gosec:
	@echo "Running gosec..."
	go run github.com/securego/gosec/v2/cmd/gosec@latest --exclude G404,G101,G115,G402 --exclude-generated -terse ./...

generate-mocks:
	find internal -type f -name "mock_*.go" -delete
	go run github.com/vektra/mockery/v2 --config ./.configs/mockery.yaml
	find internal -type f -name "mock_*.go" -exec go run mvdan.cc/gofumpt@latest -w {} \;
docker-up:
	docker compose up -d

local: docker-up
	go run cmd/euthanaisa/main.go

linux-binary:
	GOOS=linux GOARCH=amd64 go build -o bin/euthanaisa cmd/euthanaisa/main.go

test:
	go test -cover ./...

test-coverage:
	go test -coverprofile=coverage.out ./...
	go tool cover -html=coverage.out

check: fmt vulncheck deadcode gosec staticcheck helm-lint

fmt:
	@echo "Running go fmt..."
	go fmt ./...

staticcheck:
	@echo "Running staticcheck..."
	go tool honnef.co/go/tools/cmd/staticcheck -f=stylish ./...

vulncheck:
	@echo "Running vulncheck..."
	go tool golang.org/x/vuln/cmd/govulncheck ./...

deadcode:
	@echo "Running deadcode..."
	go tool golang.org/x/tools/cmd/deadcode ./...

gosec:
	@echo "Running gosec..."
	go tool github.com/securego/gosec/v2/cmd/gosec --exclude-generated -terse ./...

helm-lint:
	@echo "Running helm lint..."
	helm lint --strict ./charts

generate: generate-mocks

generate-mocks:
	find internal -type f -name "mock_*.go" -delete
	go run github.com/vektra/mockery/v2 --config ./.configs/mockery.yaml
	find internal -type f -name "mock_*.go" -exec go tool mvdan.cc/gofumpt -w {} \;

local:
	go run cmd/euthanaisa/main.go -log-level=debug -pushgateway-url=https://prometheus-pushgateway.dev.dev-nais.cloud.nais.io

linux-binary:
	GOOS=linux GOARCH=amd64 go build -o bin/app cmd/euthanaisa/main.go
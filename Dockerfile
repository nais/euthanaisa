ARG GO_VERSION=1.25
FROM golang:${GO_VERSION} AS builder
WORKDIR /src
COPY go.* /src/
RUN go mod download
COPY . /src
RUN go build -o bin/euthanaisa ./cmd/euthanaisa

FROM gcr.io/distroless/base
WORKDIR /app
COPY --from=builder /src/bin/euthanaisa /app/euthanaisa
ENTRYPOINT ["/app/euthanaisa"]

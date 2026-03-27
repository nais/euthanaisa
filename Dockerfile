FROM golang:1.26.1 AS builder
WORKDIR /src
COPY go.* /src/
RUN go mod download
COPY . /src
RUN go build -o bin/euthanaisa ./cmd/euthanaisa

FROM gcr.io/distroless/base
WORKDIR /app
COPY --from=builder /src/bin/euthanaisa /app/euthanaisa
ENTRYPOINT ["/app/euthanaisa"]

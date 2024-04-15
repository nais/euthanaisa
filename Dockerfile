ARG GO_VERSION=""
FROM golang:${GO_VERSION}alpine as builder
WORKDIR /src
COPY go.* /src/
RUN go mod download
COPY . /src
RUN go build -o bin/euthanaisa ./cmd/euthanaisa

FROM gcr.io/distroless/base
WORKDIR /app
COPY --from=builder /src/bin/euthanaisa /app/euthanaisa
ENTRYPOINT ["/app/euthanaisa"]

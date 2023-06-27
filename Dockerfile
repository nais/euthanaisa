FROM golang:1.20-alpine as builder
RUN apk add --no-cache git make
COPY . /src
WORKDIR /src
RUN rm -f go.sum
RUN go get ./...
RUN make linux-binary

FROM gcr.io/distroless/base
WORKDIR /app
COPY --from=builder /src/bin/app /app/binary
ENTRYPOINT ["/app/binary"]
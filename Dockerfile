FROM golang:1.23 AS builder

WORKDIR /app

COPY go.mod .
COPY go.sum .
COPY main.go .
RUN go mod download && \
    CGO_ENABLED=0 GOOS=linux go build -a -installsuffix cgo -o app .

FROM scratch

COPY --from=builder /app/app /fah_exporter

ENTRYPOINT ["/fah_exporter"]

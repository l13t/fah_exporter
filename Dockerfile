FROM golang:1.23-alpine AS builder

WORKDIR /app

COPY go.mod .
COPY go.sum .
COPY main.go .
RUN go mod download && \
    CGO_ENABLED=0 GOOS=linux go build -a -installsuffix cgo -o app .
RUN apk update && apk add ca-certificates && rm -rf /var/cache/apk/*

FROM scratch

COPY --from=builder /app/app /fah_exporter
COPY --from=builder /etc/ssl/ /etc/ssl/

ENTRYPOINT ["/fah_exporter"]

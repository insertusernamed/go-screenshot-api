FROM golang:1.25-alpine AS builder
WORKDIR /app
ENV CGO_ENABLED=0 \
    GOOS=linux \
    GOARCH=amd64
COPY go.mod go.sum ./
RUN go mod download
COPY . .
RUN go build -trimpath -ldflags="-s -w" -o myapp .
FROM alpine:3.20 AS runtime
WORKDIR /app
RUN apk add --no-cache ca-certificates
RUN addgroup -S appuser && adduser -S -G appuser -H -s /sbin/nologin appuser
COPY --from=builder --chown=appuser:appuser /app/myapp /app/myapp
USER appuser
ENTRYPOINT ["/app/myapp"]

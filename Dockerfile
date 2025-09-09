FROM golang:1.19-alpine AS builder
WORKDIR /app
COPY go.mod go.sum ./
RUN go mod download
COPY . .
RUN CGO_ENABLED=0 GOOS=linux go build -a -ldflags="-w -s" -o /main .
FROM alpine:latest
RUN apk --no-cache add chromium
WORKDIR /root/
COPY --from=builder /main .
EXPOSE 8083
CMD ["./main"]
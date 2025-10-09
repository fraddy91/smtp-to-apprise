FROM golang:1.25-alpine AS builder
WORKDIR /app
# Cache modules
COPY go.mod go.sum ./
RUN go mod download
COPY . .
RUN go build -o shim ./cmd/smtprise/main.go

FROM alpine:latest
WORKDIR /app
COPY --from=builder /app/shim .
CMD ["./shim"]
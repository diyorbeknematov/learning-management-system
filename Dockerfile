# Stage 1: build
FROM golang:1.27-alpine AS builder

WORKDIR /app

# Dependencies first, so this layer is cached until go.mod/go.sum change.
COPY go.mod go.sum ./
RUN go mod download

COPY . .
RUN CGO_ENABLED=0 GOOS=linux go build -trimpath -ldflags="-s -w" -o /lms ./cmd/main.go

# Stage 2: runtime
FROM alpine:3.22

# ca-certificates: TLS to SMTP/MinIO; tzdata: time zones.
RUN apk --no-cache add ca-certificates tzdata \
    && adduser -D -H -u 10001 app

COPY --from=builder /lms /lms

USER app

# HTTP_PORT defaults to 8080.
EXPOSE 8080

ENTRYPOINT ["/lms"]

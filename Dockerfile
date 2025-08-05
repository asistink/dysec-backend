# --- Tahap 1: Build ---
FROM golang:1.22-alpine AS builder
WORKDIR /app
COPY go.mod go.sum ./
RUN go mod download
COPY . .
RUN CGO_ENABLED=0 GOOS=linux go build -o /app/main ./cmd/api

# --- Tahap 2: Final Image ---
FROM alpine:latest
WORKDIR /app
COPY --from=builder /app/main .
COPY configs/ ./configs/
EXPOSE 8080
CMD ["./main"], "-b"]
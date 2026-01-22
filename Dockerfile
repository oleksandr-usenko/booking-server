# Build stage
FROM golang:1.21-alpine AS builder

WORKDIR /usr/src/app

# Copy go mod files
COPY go.mod go.sum ./
RUN go mod download

# Copy source code
COPY . .

# Build the application
RUN CGO_ENABLED=0 GOOS=linux go build -o /usr/src/app/server .

# Runtime stage
FROM alpine:latest

WORKDIR /usr/src/app

# Install ca-certificates for HTTPS
RUN apk --no-cache add ca-certificates

# Copy the binary from builder
COPY --from=builder /usr/src/app/server .

# Expose port
EXPOSE 8080

# Run the application
CMD ["./server"]

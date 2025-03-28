FROM golang:1.21-alpine AS builder

WORKDIR /app

# Copy go mod and sum files
COPY go.mod ./

# Download dependencies
RUN go mod download

# Copy source code
COPY . .

# Build the application
RUN CGO_ENABLED=0 GOOS=linux go build -o adaptive-metrics

# Use a minimal alpine image for the final stage
FROM alpine:3.18

WORKDIR /app

# Install necessary runtime dependencies
RUN apk --no-cache add ca-certificates tzdata

# Copy binary from builder stage
COPY --from=builder /app/adaptive-metrics /app/
# Copy config directories
COPY --from=builder /app/configs /app/configs

# Set environment variables
ENV CONFIG_PATH=/app/configs

# Expose API port
EXPOSE 8080

# Run the application
CMD ["/app/adaptive-metrics"]
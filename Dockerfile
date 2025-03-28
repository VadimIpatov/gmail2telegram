FROM golang:1.24-alpine AS builder

WORKDIR /app

# Copy go mod and sum files
COPY go.mod go.sum ./

# Download dependencies
RUN go mod download

# Copy source code
COPY . .

# Build the application
RUN CGO_ENABLED=0 GOOS=linux go build -o gmail2telegram

# Use a minimal alpine image for the final stage
FROM alpine:latest

WORKDIR /app

# Copy the binary from builder
COPY --from=builder /app/gmail2telegram .
COPY --from=builder /app/config.yaml .

# Create a non-root user
RUN adduser -D appuser
USER appuser

# Run the application
CMD ["./gmail2telegram"] 
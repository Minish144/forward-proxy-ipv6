# Stage 1: Builder
FROM golang:1.23-alpine AS builder

WORKDIR /app

COPY go.mod go.sum ./
RUN go mod download

COPY . .

# Build the Go application
# CGO_ENABLED=0 is important for static linking, making the binary independent of system libraries
# -o main specifies the output filename
RUN CGO_ENABLED=0 GOOS=linux go build -a -installsuffix cgo -o main .

# Stage 2: Runner
FROM alpine:latest

WORKDIR /root/

# Copy the compiled binary from the builder stage
COPY --from=builder /app/main .

# Expose the port your application listens on (if applicable)
EXPOSE 3128

# Command to run the application
CMD ["./main"]

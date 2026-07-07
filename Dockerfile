# Stage 1: Build the hd-driver-slacksocket binary
FROM golang:1.25.10 AS builder

WORKDIR /build

# Cache dependencies
COPY go.mod go.sum ./
RUN go mod download

# Build the binary
COPY . .
RUN CGO_ENABLED=0 go build -o hd-driver-slacksocket ./cmd/hd-driver-slacksocket

# Stage 2: Create the final image using honeydipper base
FROM honeydipper/honeydipper:3.10.0

# Copy the driver binary to the same directory as other built-in drivers
COPY --from=builder /build/hd-driver-slacksocket /opt/honeydipper/drivers/builtin/hd-driver-slacksocket

# Ensure the binary is executable
RUN chmod +x /opt/honeydipper/drivers/builtin/hd-driver-slacksocket

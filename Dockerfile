 Stage 1: build the Go binary
FROM golang:1.25.3 AS builder

WORKDIR /app
COPY . .

# Compile statically (important for minimal runtime)
RUN go mod tidy && go build -o gokeybaord gokeyboard.go

# Stage 2: lightweight runtime
FROM debian:bookworm-slim

# Needed for reading /dev/input and using keylogger
RUN apt-get update && apt-get install -y \
    libevdev2 udev ca-certificates && \
    rm -rf /var/lib/apt/lists/*

WORKDIR /app
COPY --from=builder /app/gokeyboard .
COPY config.yaml .

# Run the binary
CMD ["./gokeyboard"]

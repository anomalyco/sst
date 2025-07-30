FROM golang:1.23.4 AS builder
WORKDIR /app
COPY go.mod go.sum ./
RUN go mod download
COPY . .
ARG TARGETARCH
RUN CGO_ENABLED=0 GOOS=linux GOARCH=${TARGETARCH} go build -o main

# --- Runtime for amd64 architecture ---
FROM alpine AS runtime-amd64
LABEL org.opencontainers.image.source=https://github.com/sst/sst
WORKDIR /app
COPY --from=builder /app/main /app/main
CMD ["/app/main"]

# --- Runtime for arm64 architecture ---
FROM arm64v8/alpine AS runtime-arm64
LABEL org.opencontainers.image.source=https://github.com/sst/sst
WORKDIR /app
COPY --from=builder /app/main /app/main
CMD ["/app/main"]

# --- Final selector ---
FROM runtime-${TARGETARCH} AS final
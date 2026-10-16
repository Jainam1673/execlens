# Multi-stage Dockerfile for ExecLens
FROM node:20-slim AS frontend-builder
WORKDIR /app/frontend
COPY frontend/package*.json ./
RUN npm ci
COPY frontend/ ./
RUN npm run build

FROM golang:1.24-bookworm AS server-builder
WORKDIR /app
COPY go.mod go.sum ./
RUN go mod download
COPY . .
RUN go build -trimpath -o /app/bin/execlens ./cmd/execlens

# Final runtime environment with all 6 language toolchains
FROM ubuntu:24.04

ENV DEBIAN_FRONTEND=noninteractive
ENV PATH="/usr/local/go/bin:/root/.cargo/bin:$PATH"

RUN apt-get update && apt-get install -y --no-install-recommends \
    build-essential \
    g++ \
    gcc \
    curl \
    git \
    openjdk-21-jdk-headless \
    python3 \
    python3-minimal \
    nodejs \
    npm \
    ca-certificates \
    && rm -rf /var/lib/apt/lists/*

# Install Rust
RUN curl --proto '=https' --tlsv1.2 -sSf https://sh.rustup.rs | sh -s -- -y --profile minimal

# Install Go
RUN curl -fsSL https://go.dev/dl/go1.24.0.linux-amd64.tar.gz | tar -C /usr/local -xz

WORKDIR /app

COPY --from=server-builder /app/bin/execlens /app/execlens
COPY --from=frontend-builder /app/frontend/dist /app/frontend/dist
COPY benchmarks /app/benchmarks
COPY migrations /app/migrations

EXPOSE 8080

ENTRYPOINT ["/app/execlens"]
CMD ["server", "--host", "0.0.0.0", "--port", "8080", "--db", "/app/data/execlens.db", "--benchmarks", "/app/benchmarks", "--frontend", "/app/frontend"]

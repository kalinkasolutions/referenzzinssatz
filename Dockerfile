# Builder and runtime share a Debian release: the CGO build for SQLite links
# against glibc and fails to start on an older one.
FROM golang:1.26-bookworm AS builder
WORKDIR /app

COPY go.mod go.sum ./
RUN go mod download
COPY . .
# The release workflow passes the tag and commit; a plain `docker build` keeps the defaults in main.go.
ARG VERSION="development build"
ARG COMMIT_SHA="unknown"
RUN CGO_ENABLED=1 go build -trimpath \
    -ldflags="-s -w -X 'main.version=${VERSION}' -X 'main.commit=${COMMIT_SHA}'" \
    -o referenzzinssatz .

FROM debian:bookworm-slim
RUN set -eux; \
    apt-get update; \
    DEBIAN_FRONTEND=noninteractive apt-get install -y --no-install-recommends \
        ca-certificates \
        chromium; \
    rm -rf /var/lib/apt/lists/*

# Chromium runs with --no-sandbox, so the process must not run as root.
RUN useradd --system --uid 10001 --create-home app \
    && mkdir -p /app/data \
    && chown app:app /app/data

WORKDIR /app
COPY --from=builder /app/referenzzinssatz .
USER app
CMD ["./referenzzinssatz"]

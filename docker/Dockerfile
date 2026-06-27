# ── Build stage ───────────────────────────────────────────────────────────────
# Pinned to the toolchain in go.mod (1.25.x) for reproducible builds.
# CGO is disabled: the result is a fully static binary that runs on a bare Alpine.
FROM golang:1.25-alpine AS builder

WORKDIR /src
COPY src/go.mod src/go.sum ./
RUN go mod download
COPY src/ .
RUN CGO_ENABLED=0 GOOS=linux go build -trimpath -ldflags="-s -w" -o /lafourche ./cmd/lafourche

# ── Runtime stage ─────────────────────────────────────────────────────────────
# Authentication is a headless Firebase email/password login — no browser at
# runtime. All that's needed is a CA bundle for TLS.
FROM alpine:3.21

RUN apk add --no-cache ca-certificates tzdata \
    && adduser -D -u 10001 app

COPY --from=builder /lafourche /usr/local/bin/lafourche
COPY docker/entrypoint.sh /entrypoint.sh
RUN chmod +x /entrypoint.sh

# MCP HTTP on :8080, and session (auth tokens + cart id) under the mounted /data
# volume. XDG_CONFIG_HOME drives os.UserConfigDir() → /data/lafourche.
ENV MCP_ADDR=:8080 \
    XDG_CONFIG_HOME=/data

RUN mkdir -p /data/lafourche && chown -R app:app /data
USER app
VOLUME ["/data"]

# MCP Streamable HTTP
EXPOSE 8080

ENTRYPOINT ["/entrypoint.sh"]

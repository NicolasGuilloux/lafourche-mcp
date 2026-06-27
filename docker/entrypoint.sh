#!/bin/sh
# Headless entrypoint for the La Fourche MCP server.
#
# Boot sequence:
#   1. log in (Firebase email/password, no browser) if LAFOURCHE_EMAIL /
#      LAFOURCHE_PASSWORD are provided,
#   2. exec the requested command (default: the MCP HTTP server).
#
# Auth tokens and the cart id live on the mounted /data volume, so logging in
# once persists across restarts. Unauthenticated commands (search) need nothing.
#
# Any extra arguments passed to the container override step 2, e.g.
#   docker compose run --rm lafourche search "lait d'amande"
set -e

# ── Authentication (optional, headless) ───────────────────────────────────────
if [ -n "${LAFOURCHE_EMAIL}" ] && [ -n "${LAFOURCHE_PASSWORD}" ]; then
    echo "Logging in as ${LAFOURCHE_EMAIL}..."
    lafourche login || \
        echo "WARN: login failed — authenticated commands (orders, basket, info) will error until you log in." >&2
fi

# ── Command (default: MCP HTTP server) ─────────────────────────────────────────
if [ "$#" -eq 0 ]; then
    set -- mcp http "${MCP_ADDR:-:8080}"
fi

exec lafourche "$@"

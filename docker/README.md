# Docker

A single static Go binary (~15 MB image). Authentication is a headless Firebase
email/password login — no browser at runtime.

```bash
cp .env.example .env          # fill in LAFOURCHE_EMAIL / LAFOURCHE_PASSWORD
docker compose up --build     # MCP Streamable HTTP on :8080
docker compose run --rm lafourche search "lait d'amande"
docker compose run --rm lafourche orders
```

## Authentication

Two ways to get an authenticated session into the container:

### a) Auto-login (recommended)

Set credentials in `.env`; the entrypoint logs in on start and persists the
tokens to the volume:

```env
LAFOURCHE_EMAIL=you@example.com
LAFOURCHE_PASSWORD=••••••••
```

### b) Reuse a host session

Point `LAFOURCHE_CONFIG_DIR` at a host config dir that has already run
`lafourche login`:

```env
# macOS
LAFOURCHE_CONFIG_DIR=$HOME/Library/Application Support/lafourche
# Linux
# LAFOURCHE_CONFIG_DIR=$HOME/.config/lafourche
```

The container then shares that host's `session.json` (Firebase tokens + cart
id), so `orders` / `basket` / `info` work immediately.

## One-shot commands

```bash
docker run --rm \
  -e LAFOURCHE_EMAIL=you@example.com -e LAFOURCHE_PASSWORD=secret \
  -v lafourche-config:/data/lafourche \
  ghcr.io/nicolasguilloux/lafourche-mcp:latest search "chocolat noir"
```

## Build locally

```bash
docker build -t lafourche-mcp:latest -f docker/Dockerfile .
```

The session (tokens + cart id) lives under `/data/lafourche` inside the
container (`XDG_CONFIG_HOME=/data`). Mount `/data` (or `/data/lafourche`) to
persist it.

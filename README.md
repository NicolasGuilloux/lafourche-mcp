# lafourche-mcp

[![build](https://github.com/NicolasGuilloux/lafourche-mcp/actions/workflows/build.yml/badge.svg)](https://github.com/NicolasGuilloux/lafourche-mcp/actions/workflows/build.yml)
[![docker](https://github.com/NicolasGuilloux/lafourche-mcp/actions/workflows/docker.yml/badge.svg)](https://github.com/NicolasGuilloux/lafourche-mcp/actions/workflows/docker.yml)

CLI (and MCP server) for [La Fourche](https://lafourche.fr), the French organic
grocery store. Log in, search products, manage your basket, and view your order
history by talking to the La Fourche back-ends. Authentication is a **headless
Firebase email/password login** — no browser anywhere. The basket is your
**account cart**, the very same one as on the website and the mobile app
(see [How it works](#how-it-works)).

> [!WARNING]
> **Disclaimer — this application was vibecoded.** It was built largely through
> AI-assisted, exploratory "vibe coding" rather than a formal engineering
> process. Expect rough edges. It is not affiliated with, endorsed by, or
> supported by La Fourche. Use it at your own risk and in accordance with the
> relevant terms of service.

## CLI commands

| Command | Description |
| --- | --- |
| `login` | Log in with email/password (Firebase, no browser). |
| `logout` | Log out and clear stored tokens. |
| `info` | Show the logged-in account info. |
| `search <query>` | Search the product catalog (50 results/page, `--page N`). |
| `orders` | List your previous orders. |
| `orders <number>` | Show one order with all its articles. |
| `basket get` | Show the current account basket. |
| `basket add <sku> [qty]` | Add a product by SKU (default qty 1). |
| `basket remove <sku> [qty]` | Remove a product by SKU (default: all). |
| `mcp` | Run the MCP server over stdio. |
| `mcp http [addr]` | Run the MCP server over Streamable HTTP (default `:8080`). |

`search` works without logging in. `info`, `orders`, and `basket` require being
logged in. Global flag `--format text|json` (default `text`).

## MCP tools

When run as an MCP server (`mcp` / `mcp http`), the following tools are exposed:

| Tool | Arguments | Description |
| --- | --- | --- |
| `search_products` | `query` (required), `page`, `size` | Search the product catalog (50/page). Returns SKUs used by the basket. |
| `user_info` | — | Get the connected account info. Requires login. |
| `basket_get` | — | Get current account basket contents. Requires login. |
| `basket_add` | `sku` (required), `quantity` | Add a product to the basket by SKU. Requires login. |
| `basket_remove` | `sku` (required), `quantity` | Remove a product from the basket (0 = remove all). Requires login. |
| `orders` | `limit` | List recent orders. Requires login. |
| `order_detail` | `number` (required) | Get a specific order with all its articles. Requires login. |

Login itself is not an MCP tool — authenticate once with the `login` CLI command
(or by reusing a host session), then point your MCP client at the server.

## Run it

Pick one of the three ways below. Authenticated features need an account —
provide `LAFOURCHE_EMAIL` / `LAFOURCHE_PASSWORD`, or run `login` once.

> [!IMPORTANT]
> **Log in first for authenticated features.** `info`, `orders`, and `basket`
> require a valid session. Run `login` once (or set `LAFOURCHE_EMAIL` /
> `LAFOURCHE_PASSWORD`); tokens are stored in the config dir and refreshed
> automatically. The MCP server has no login tool.

### A. From sources (Go)

Tooling is provided by **Nix devenv** (Go, golangci-lint, …):

```bash
direnv allow                       # or: devenv shell
cd src && go build -o ../bin/lafourche ./cmd/lafourche
cd src && go test ./...            # optional

./bin/lafourche login              # email/password, headless
./bin/lafourche search "lait d'amande"
./bin/lafourche basket add 1-NTV-207 2
./bin/lafourche basket get
./bin/lafourche mcp http :8080     # MCP server on :8080
```

Everything runs headless — no browser at any point.

### B. Docker (CLI + MCP)

Pull the prebuilt image from GitHub Container Registry, then run one-shot
commands or the MCP server. The `-v` bind-mount persists the session (tokens +
cart id).

```bash
docker pull ghcr.io/nicolasguilloux/lafourche-mcp:latest

# one-shot command
docker run --rm \
  -e LAFOURCHE_EMAIL=you@example.com -e LAFOURCHE_PASSWORD=secret \
  -v lafourche-config:/data/lafourche \
  ghcr.io/nicolasguilloux/lafourche-mcp:latest search "chocolat noir"

# MCP server (no args → `mcp http :8080`)
docker run --rm -p 8080:8080 \
  -e LAFOURCHE_EMAIL=you@example.com -e LAFOURCHE_PASSWORD=secret \
  -v lafourche-config:/data/lafourche \
  ghcr.io/nicolasguilloux/lafourche-mcp:latest
```

The image is a static binary on Alpine — no browser, no system deps. Prefer
building locally? `docker build -t lafourche-mcp:latest -f docker/Dockerfile .`

### C. Docker Compose

```bash
cp .env.example .env        # fill LAFOURCHE_EMAIL / LAFOURCHE_PASSWORD
docker compose up --build                          # MCP server on :8080
docker compose run --rm lafourche search "lait d'amande"
docker compose run --rm lafourche orders
```

Set `LAFOURCHE_CONFIG_DIR` in `.env` to reuse a host session; leave it empty for
a self-contained named volume (then set the credentials). Full details in
[`docker/README.md`](docker/README.md).

## Configuration

Configured entirely through environment variables (see [`.env.example`](.env.example)):

| Variable | Used by | Purpose |
| --- | --- | --- |
| `LAFOURCHE_EMAIL` | login | Account email (auto-login / non-interactive). |
| `LAFOURCHE_PASSWORD` | login | Account password. |
| `LAFOURCHE_CONFIG_DIR` | compose | Host config dir to reuse inside the container (session). |
| `MCP_ADDR` | server | MCP HTTP listen address (default `:8080`). |
| `LAFOURCHE_SESSION_PATH` | all | Override the session file path. |
| `LAFOURCHE_MEMBER_API_URL` | orders, basket | Member GraphQL API (default `https://api.lafourche.fr/graphql`). |
| `LAFOURCHE_FIREBASE_PROJECT_ID` | basket | Firestore project (default `production-la-fourche`). |
| `LAFOURCHE_FIREBASE_API_KEY` | login | Firebase Web API key (default: the front-end key). |
| `LAFOURCHE_ALGOLIA_APP_ID` / `_API_KEY` / `_INDEX` | search | Algolia search config (defaults: the site's public values). |

State (Firebase tokens + account cart id) lives in a single `session.json` in
the config dir — `~/.config/lafourche` on Linux,
`~/Library/Application Support/lafourche` on macOS, or `/data/lafourche` in the
container (`XDG_CONFIG_HOME=/data`).

## How it works

La Fourche has two back-ends; this tool uses each for what it does best.

- **Search** → the site's **Algolia** index (`production_products`) — the very
  engine lafourche.fr uses. Same results, **member prices**, and each product's
  **SKU** (the key used by the basket).

- **Login, account, basket, orders** → the **member back-end**
  (`api.lafourche.fr` GraphQL + **Firebase/Firestore**):
  - **Authentication** is a headless **Firebase email/password** sign-in
    (`identitytoolkit.googleapis.com`), with automatic refresh via
    `securetoken.googleapis.com`. The ID token is the Bearer for both the
    member API and Firestore.
  - The **basket is your account cart**, stored in **Firestore**
    (`customers/<uid>.shoppingCartId → carts/<id> = { SKU: quantity }`). That's
    why it stays in sync with the website and the mobile app. Product names and
    **member prices** are resolved through the `createCart` GraphQL mutation.
  - **Orders** come from the `GetCustomerOrder` GraphQL query.

There is no browser at any point. Full reverse-engineering notes in
[`docs/INVESTIGATION.md`](docs/INVESTIGATION.md).

## Docs

- [`docker/README.md`](docker/README.md) — Docker specifics, session reuse
- [`docs/INVESTIGATION.md`](docs/INVESTIGATION.md) — reverse-engineered API surface
- [`CLAUDE.md`](CLAUDE.md) — repository guide for AI assistants

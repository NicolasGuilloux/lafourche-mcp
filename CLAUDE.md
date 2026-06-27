# CLAUDE.md

This file provides guidance to Claude Code (claude.ai/code) when working with
code in this repository.

## Project Overview

CLI tool **and** MCP server for the La Fourche French organic grocery store
(lafourche.fr). It lets you log in, search products, manage your account basket
and view order history by talking to the La Fourche back-ends.

## Development Environment

Uses **Nix devenv** for tooling (Go, golangci-lint, …). After cloning,
`direnv allow` or `devenv shell` sets everything up.

## Build & Test

All Go code lives under `src/` (the module root). Run Go commands from there:

```bash
cd src && go build -o ../bin/lafourche ./cmd/lafourche   # build
cd src && go test ./...                                  # run all tests
cd src && go test ./internal/format -run TestBasket      # a single test
```

Or via devenv scripts (from the repo root): `build`, `test`, `lint`, `fmt`,
`serve-stdio`, `serve-http`.

## Architecture

- **Language**: Go (module: `github.com/NicolasGuilloux/lafourche-mcp`).
- **Local state**: a single `session.json` in the config dir (Firebase tokens +
  account cart id) — `~/.config/lafourche` (Linux) /
  `~/Library/Application Support/lafourche` (macOS), overridable with
  `XDG_CONFIG_HOME` or `LAFOURCHE_SESSION_PATH`.
- **Entry point**: `src/cmd/lafourche/main.go` → `internal/cli` (cobra) and
  `internal/mcpserver` (MCP, stdio + HTTP).

## API interaction

La Fourche has two back-ends; the tool uses each for what it does best:

- **Search** → the site's **Algolia** index (`production_products`). Same results
  as lafourche.fr, member prices, and the product **SKU** used by the basket.
- **Auth, account, basket, orders** → the member back-end
  (`api.lafourche.fr` GraphQL + **Firebase/Firestore**):
  - **Login** is a headless **Firebase email/password** sign-in
    (`identitytoolkit.googleapis.com`), refreshed via `securetoken.googleapis.com`.
    The ID token is the Bearer for both `api.lafourche.fr` and Firestore.
  - **Basket** is the **account cart**, stored in Firestore
    (`customers/<uid>.shoppingCartId → carts/<id> = { SKU: qty }`) and therefore
    synced with the website and the mobile app. Names + member prices are
    enriched via the `createCart` GraphQL mutation (amounts in cents).
  - **Orders** come from the `GetCustomerOrder` GraphQL query.

Reverse-engineering notes: `docs/INVESTIGATION.md`.

## Code layout

```
src/internal/cli/         cobra commands
src/internal/mcpserver/   MCP server (mark3labs/mcp-go): stdio + HTTP
src/internal/format/      shared rendering: Markdown (for LLMs) + JSON
src/internal/lafourche/   back-end client
  auth.go        Firebase email/password login
  member.go      orders (member API)
  membercart.go  account basket (Firestore + createCart enrichment)
  firestore.go   Firestore REST access (cart)
  info.go        account info (token claims)
  search.go      product search (Algolia, index production_products)
```

## Adding a command / MCP tool

- CLI: add a `*cobra.Command` in `src/internal/cli/cli.go`.
- MCP: add an `s.AddTool(...)` in `src/internal/mcpserver/server.go`.
- Logic: a method on `*lafourche.Client` in `src/internal/lafourche`.
- Rendering: helpers in `src/internal/format`.

## Conventions

- Keep `gofmt`/`go vet` clean and add table-driven tests where it helps.
- Authenticated operations call `ensureToken` (auto-refresh) and return
  `ErrNotAuthenticated` when no valid session exists.

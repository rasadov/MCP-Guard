# MCP Guard

Secure MCP Gateway prototype with policy enforcement, audit logging, and a governance dashboard. Agents connect via MCP (HTTP); tool calls are proxied to a Slack MCP server with least-privilege controls.

## Quick Start

```bash
cp .env.example .env
docker compose up --build
```

Open http://localhost:8080 and log in via dev auth:

http://localhost:8080/auth/dev-login?email=admin@mcpguard.local

## Demo API Key

Seed creates a demo agent key:

```
mcpg_demo_7f3a9b2c1d4e5f6a8b9c0d1e2f3a4b5c
```

## Gemini CLI

```bash
gemini mcp add --transport http --header "Authorization: Bearer mcpg_demo_7f3a9b2c1d4e5f6a8b9c0d1e2f3a4b5c" mcp-guard http://localhost:8080/mcp
```

See [scripts/demo-gemini.md](scripts/demo-gemini.md) for the full walkthrough.

## Architecture

- **Gateway** (Go + Gin): MCP server for agents, REST API for dashboard
- **Slack MCP** (stdio subprocess): proxied downstream connector
- **PostgreSQL**: users, agents, skills, policies, audit logs

See [docs/architecture.md](docs/architecture.md).

## Development

```bash
# Backend
go run ./cmd/gateway

# Frontend
cd web && bun install && bun run build
```

## Security

See [docs/security.md](docs/security.md).

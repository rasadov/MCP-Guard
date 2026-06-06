# MCP Guard

Secure MCP Gateway prototype with policy enforcement, audit logging, and a governance dashboard. Agents connect via MCP (HTTP); tool calls are proxied to a Slack MCP server with least-privilege controls.

## Quick Start

```bash
cp .env.example .env
docker compose up --build
```

Open http://localhost:8080 and sign in:

- **Production:** Sign in with Google (configure `GOOGLE_CLIENT_ID` / `GOOGLE_CLIENT_SECRET`)
- **Local dev:** Use the login page dev login with `admin@mcpguard.local` when `AUTH_DEV_MODE=true`

## Agent API keys

API keys are not seeded. After signing in:

1. Open **Agents** in the dashboard
2. Create an agent (optionally assign a skill)
3. Copy the one-time API key shown after creation

Use that key with MCP clients:

```bash
gemini mcp add --transport http --header "Authorization: Bearer <your_api_key>" mcp-guard http://localhost:8080/mcp
```

See [scripts/demo-gemini.md](scripts/demo-gemini.md) for a full walkthrough.

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

# Gemini CLI Demo Script

## Prerequisites

1. MCP Guard running: `docker compose up`
2. Gemini CLI installed
3. Demo API key from seed: `mcpg_demo_7f3a9b2c1d4e5f6a8b9c0d1e2f3a4b5c`

## Connect Gemini to MCP Guard

Use the **MCP Guard agent API key** (not your Gemini API key):

```
mcpg_demo_7f3a9b2c1d4e5f6a8b9c0d1e2f3a4b5c
```

```powershell
gemini mcp add --transport http --header "Authorization: Bearer mcpg_demo_7f3a9b2c1d4e5f6a8b9c0d1e2f3a4b5c" mcp-guard http://localhost:8080/mcp
```

If you see `no server available` or `Disconnected`, the demo API key is missing from the database. Restart with a fresh volume or rebuild:

```bash
docker compose down -v
docker compose up --build
```

Check gateway logs for `demo credentials restored` or `seed data created`.

## Demo Flow

### 1. Allowed read action

Ask Gemini:

> Read recent messages from our Slack general channel and summarize them.

Expected: `slack.conversations_history` allowed, audit log shows `outcome=allowed`.

### 2. Blocked write (read-only skill)

Ask Gemini:

> Post the summary to #general.

Expected: denied with `skill_denied` or `policy_denied`, visible in dashboard audit feed.

### 3. Allowed write (poster skill)

In dashboard, assign agent `gemini-demo` to `marketing-poster` skill, then retry post to allowed channel.

Expected: `slack.conversations_add_message` allowed for channel `C00000000`.

### 4. Blocked dangerous action

Ask Gemini:

> Create a new Slack usergroup called "ops-admins".

Expected: denied via default policy (`slack.usergroups_create`).

### 5. Shadow AI flag

```bash
curl -X POST http://localhost:8080/api/v1/shadow-events ^
  -H "Cookie: mcp_guard_token=<admin-jwt>" ^
  -H "Content-Type: application/json" ^
  -d "{\"agent_name\":\"gemini-demo\",\"tool_name\":\"slack.conversations_add_message\",\"source\":\"direct-slack-api\"}"
```

Open `/shadow` in dashboard to see the flag when no matching gateway audit exists.

### 6. Export audit logs

Dashboard → Audit → Export JSON/CSV, or:

```bash
curl "http://localhost:8080/api/v1/audit/export?format=csv" -H "Cookie: mcp_guard_token=<token>" -o audit.csv
```

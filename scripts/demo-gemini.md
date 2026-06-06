# Gemini CLI Demo Script

## Prerequisites

1. MCP Guard running: `docker compose up`
2. Gemini CLI installed
3. Dashboard login (dev login or Google OAuth)

## Create an API key

1. Open http://localhost:8080/login and sign in
2. Go to **Agents** → **Create agent & key**
3. Copy the one-time API key (it will not be shown again)

## Connect Gemini to MCP Guard

Use the **MCP Guard agent API key** from the dashboard (not your Gemini API key):

```powershell
gemini mcp add --transport http --header "Authorization: Bearer <your_api_key>" mcp-guard http://localhost:8080/mcp
```

If you see `no server available` or `Disconnected`, verify the agent exists and the API key is correct. Create a new key from the Agents page if needed.

## Demo Flow

### 1. Allowed read action

Ask Gemini:

> Read recent messages from our Slack general channel and summarize them.

Expected: `slack.conversations_history` allowed, audit log shows `outcome=allowed`.

### 2. Blocked write (read-only skill)

Create an agent with the **Marketing Readonly** skill, then ask Gemini:

> Post the summary to #general.

Expected: denied with `skill_denied` or `policy_denied`, visible in dashboard audit feed.

### 3. Allowed write (poster skill)

In dashboard **Governance**, assign your agent to the `marketing-poster` skill, then retry post to allowed channel.

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
  -d "{\"agent_name\":\"my-agent\",\"tool_name\":\"slack.conversations_add_message\",\"source\":\"direct-slack-api\"}"
```

Open `/shadow` in dashboard to see the flag when no matching gateway audit exists.

### 6. Export audit logs

Dashboard → Audit → Export JSON/CSV, or:

```bash
curl "http://localhost:8080/api/v1/audit/export?format=csv" -H "Cookie: mcp_guard_token=<token>" -o audit.csv
```

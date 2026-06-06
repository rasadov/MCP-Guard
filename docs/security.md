# Security Considerations

## Authentication

- **Agents**: API keys (`mcpg_<prefix>_<secret>`), bcrypt-hashed in database
- **Dashboard**: Dev login (`AUTH_DEV_MODE=true`)
- **Sessions**: JWT in HttpOnly cookie

## Authorization (RBAC)

- `admin`: manage policies, skills, shadow events, full audit export
- `user`: own agents, read audit, discover tools

## Policy Defaults

- Fail closed: missing key, unknown agent, or policy error → deny + audit
- Default deny on dangerous Slack tools (usergroups, reactions)
- Write tools blocked for `user` role unless policy allows
- Channel allowlist for `conversations_add_message`

## Audit

- Parameters sanitized: tokens/secrets redacted, long strings truncated
- Export available as JSON/CSV for compliance review

## Token Handling

- `SLACK_BOT_TOKEN` only passed to Slack MCP subprocess environment
- Never logged or returned in API responses
- `.env` excluded from git

## Rate Limiting (recommended for production)

Add Gin middleware (~100 req/min per API key) before public deployment.

## Input Validation

- Policy/skill JSON validated on write via API binding
- Tool parameters forwarded as-is to Slack MCP after policy check

## Deployment

- Rotate `JWT_SECRET` and agent API keys in production
- Disable `AUTH_DEV_MODE` and `SEED_ON_START`
- Use TLS termination (reverse proxy or load balancer)

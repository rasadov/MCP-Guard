FROM oven/bun:1-alpine AS web-build
WORKDIR /app/web
COPY web/package.json web/bun.lock* ./
RUN bun install --frozen-lockfile || bun install
COPY web/ ./
RUN bun run build

FROM golang:1.26-bookworm AS go-build
WORKDIR /app
COPY go.mod go.sum ./
RUN go mod download
COPY . .
COPY --from=web-build /app/web/dist ./web/dist
RUN CGO_ENABLED=0 go build -o /gateway ./cmd/gateway

FROM debian:bookworm-slim
RUN apt-get update && apt-get install -y --no-install-recommends ca-certificates curl && rm -rf /var/lib/apt/lists/*
WORKDIR /app

ARG SLACK_MCP_VERSION=v1.3.0
RUN curl -fsSL -o /usr/local/bin/slack-mcp-server \
  "https://github.com/korotovsky/slack-mcp-server/releases/download/${SLACK_MCP_VERSION}/slack-mcp-server-linux-amd64" \
  && chmod +x /usr/local/bin/slack-mcp-server

COPY --from=go-build /gateway /app/gateway
ENV SLACK_MCP_PATH=/usr/local/bin/slack-mcp-server
EXPOSE 8080
CMD ["/app/gateway"]

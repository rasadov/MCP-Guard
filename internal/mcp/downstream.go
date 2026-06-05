package mcpgw

import (
	"bytes"
	"context"
	"fmt"
	"log/slog"
	"os"
	"os/exec"
	"strings"
	"sync"

	"github.com/modelcontextprotocol/go-sdk/mcp"
	"github.com/rasadov/mcp-guard/internal/config"
)

const toolPrefix = "slack."

type Downstream struct {
	mu       sync.RWMutex
	session  *mcp.ClientSession
	tools    map[string]*mcp.Tool
	rawNames map[string]string
}

func ConnectSlack(ctx context.Context, cfg config.Config) (*Downstream, error) {
	if cfg.Slack.BotToken == "" {
		slog.Warn("SLACK_BOT_TOKEN not set; downstream slack tools unavailable")
		return &Downstream{tools: map[string]*mcp.Tool{}, rawNames: map[string]string{}}, nil
	}

	slackEnv, err := slackMCPEnv(cfg.Slack.BotToken)
	if err != nil {
		slog.Warn("SLACK_BOT_TOKEN invalid for slack-mcp-server", "error", err)
		return &Downstream{tools: map[string]*mcp.Tool{}, rawNames: map[string]string{}}, nil
	}

	cmd := exec.CommandContext(ctx, cfg.SlackMCPPath)
	cmd.Env = append(os.Environ(), slackEnv...)
	var stderr bytes.Buffer
	cmd.Stderr = &stderr

	client := mcp.NewClient(&mcp.Implementation{Name: "mcp-guard-proxy", Version: "0.1.0"}, nil)
	transport := &mcp.CommandTransport{Command: cmd}
	session, err := client.Connect(ctx, transport, nil)
	if err != nil {
		if msg := strings.TrimSpace(stderr.String()); msg != "" {
			return nil, fmt.Errorf("connect slack mcp: %w: %s", err, summarizeSlackStderr(msg))
		}
		return nil, fmt.Errorf("connect slack mcp: %w", err)
	}

	result, err := session.ListTools(ctx, nil)
	if err != nil {
		_ = session.Close()
		return nil, fmt.Errorf("list slack tools: %w", err)
	}

	ds := &Downstream{
		session:  session,
		tools:    make(map[string]*mcp.Tool, len(result.Tools)),
		rawNames: make(map[string]string, len(result.Tools)),
	}
	for _, tool := range result.Tools {
		name := toolPrefix + tool.Name
		copyTool := *tool
		copyTool.Name = name
		ds.tools[name] = &copyTool
		ds.rawNames[name] = tool.Name
	}
	slog.Info("slack mcp connected", "tools", len(ds.tools))
	return ds, nil
}

func (d *Downstream) Tools() map[string]*mcp.Tool {
	d.mu.RLock()
	defer d.mu.RUnlock()
	out := make(map[string]*mcp.Tool, len(d.tools))
	for k, v := range d.tools {
		copyTool := *v
		out[k] = &copyTool
	}
	return out
}

func (d *Downstream) Call(ctx context.Context, gatewayName string, args map[string]any) (*mcp.CallToolResult, error) {
	d.mu.RLock()
	session := d.session
	rawName := d.rawNames[gatewayName]
	d.mu.RUnlock()

	if session == nil || rawName == "" {
		return nil, fmt.Errorf("tool %s unavailable", gatewayName)
	}
	return session.CallTool(ctx, &mcp.CallToolParams{
		Name:      rawName,
		Arguments: args,
	})
}

func summarizeSlackStderr(stderr string) string {
	if strings.Contains(stderr, "Failed to fetch users") || strings.Contains(stderr, "RefreshUsers") {
		return "slack-mcp-server missing users:read bot scope — add it, Reinstall to Workspace, update SLACK_BOT_TOKEN in .env, restart gateway"
	}
	if strings.Contains(stderr, "Failed to fetch channels") || strings.Contains(stderr, "zero channels") {
		return "slack-mcp-server missing channel scopes — add channels:read, groups:read, im:read, mpim:read (and channels:history), Reinstall to Workspace, restart gateway"
	}
	for _, line := range strings.Split(stderr, "\n") {
		line = strings.TrimSpace(line)
		if line == "" {
			continue
		}
		if strings.Contains(line, "missing_scope") {
			return "slack-mcp-server missing bot scopes — see OAuth & Permissions → Bot Token Scopes, then Reinstall to Workspace"
		}
		if strings.Contains(line, "invalid_auth") {
			return "slack-mcp-server rejected the token (check SLACK_BOT_TOKEN is a valid xoxb- bot token)"
		}
	}
	if len(stderr) > 300 {
		return stderr[:300] + "..."
	}
	return stderr
}

func slackMCPEnv(token string) ([]string, error) {
	env := []string{"SLACK_MCP_ADD_MESSAGE_TOOL=true"}
	switch {
	case strings.HasPrefix(token, "xoxb-"):
		env = append(env, "SLACK_MCP_XOXB_TOKEN="+token)
	case strings.HasPrefix(token, "xoxp-"):
		env = append(env, "SLACK_MCP_XOXP_TOKEN="+token)
	default:
		return nil, fmt.Errorf("expected xoxb- (bot) or xoxp- (user) token, not app-level xapp- tokens")
	}
	return env, nil
}

func (d *Downstream) Close() error {
	d.mu.Lock()
	defer d.mu.Unlock()
	if d.session == nil {
		return nil
	}
	err := d.session.Close()
	d.session = nil
	return err
}

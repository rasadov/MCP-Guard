package mcpgw

import (
	"context"
	"fmt"
	"log/slog"
	"os"
	"os/exec"
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

	cmd := exec.CommandContext(ctx, cfg.SlackMCPPath)
	cmd.Env = append(os.Environ(),
		"SLACK_MCP_XOXP_TOKEN="+cfg.Slack.BotToken,
		"SLACK_MCP_ADD_MESSAGE_TOOL=true",
	)

	client := mcp.NewClient(&mcp.Implementation{Name: "mcp-guard-proxy", Version: "0.1.0"}, nil)
	transport := &mcp.CommandTransport{Command: cmd}
	session, err := client.Connect(ctx, transport, nil)
	if err != nil {
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

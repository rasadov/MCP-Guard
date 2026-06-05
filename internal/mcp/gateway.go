package mcpgw

import (
	"context"
	"encoding/json"
	"fmt"
	"log/slog"
	"net/http"
	"strings"
	"sync"
	"time"

	"github.com/google/uuid"
	"github.com/modelcontextprotocol/go-sdk/mcp"
	"github.com/rasadov/mcp-guard/internal/audit"
	"github.com/rasadov/mcp-guard/internal/auth"
	"github.com/rasadov/mcp-guard/internal/models"
	"github.com/rasadov/mcp-guard/internal/policy"
	"github.com/rasadov/mcp-guard/internal/seed"
	"gorm.io/gorm"
)

type Gateway struct {
	db         *gorm.DB
	downstream *Downstream
	apiKeys    *auth.APIKeyService
	policy     *policy.Engine
	audit      *audit.Service
	sessions   sync.Map // sessionID -> *models.Agent
}

func NewGateway(db *gorm.DB, downstream *Downstream, apiKeys *auth.APIKeyService, policyEngine *policy.Engine, auditSvc *audit.Service) *Gateway {
	return &Gateway{
		db:         db,
		downstream: downstream,
		apiKeys:    apiKeys,
		policy:     policyEngine,
		audit:      auditSvc,
	}
}

func (g *Gateway) Handler() http.Handler {
	mcpHandler := mcp.NewStreamableHTTPHandler(func(req *http.Request) *mcp.Server {
		agent, err := g.authenticate(req)
		if err != nil {
			slog.Warn("mcp auth failed", "error", err, "has_auth_header", req.Header.Get("Authorization") != "")
			return nil
		}
		sessionID := req.Header.Get("Mcp-Session-Id")
		if sessionID != "" {
			g.sessions.Store(sessionID, agent)
			g.touchSession(agent)
		}
		return g.serverForAgent(agent)
	}, nil)

	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Only require auth when establishing a new MCP session (no session header yet).
		if r.Header.Get("Mcp-Session-Id") == "" {
			if _, err := g.authenticate(r); err != nil {
				http.Error(w, "unauthorized: set Authorization: Bearer <mcp_guard_api_key> (demo: "+seed.DemoAPIKey+")", http.StatusUnauthorized)
				return
			}
		}
		mcpHandler.ServeHTTP(w, r)
	})
}

func (g *Gateway) authenticate(req *http.Request) (*models.Agent, error) {
	authHeader := req.Header.Get("Authorization")
	if authHeader == "" {
		return nil, auth.ErrInvalidAPIKey
	}
	return g.apiKeys.Validate(authHeader)
}

func (g *Gateway) touchSession(agent *models.Agent) {
	now := time.Now()
	var session models.Session
	err := g.db.Where("agent_id = ?", agent.ID).First(&session).Error
	if err != nil {
		session = models.Session{AgentID: agent.ID, LastSeen: now}
		_ = g.db.Create(&session).Error
		return
	}
	_ = g.db.Model(&session).Update("last_seen", now).Error
}

func (g *Gateway) loadPolicies() ([]models.Policy, error) {
	var policies []models.Policy
	return policies, g.db.Where("enabled = ?", true).Find(&policies).Error
}

func (g *Gateway) serverForAgent(agent *models.Agent) *mcp.Server {
	server := mcp.NewServer(&mcp.Implementation{Name: "mcp-guard", Version: "0.1.0"}, nil)

	server.AddReceivingMiddleware(func(next mcp.MethodHandler) mcp.MethodHandler {
		return func(ctx context.Context, method string, req mcp.Request) (mcp.Result, error) {
			if method == "tools/list" {
				return g.filteredListTools(ctx, agent, next, method, req)
			}
			return next(ctx, method, req)
		}
	})

	for name, tool := range g.downstream.Tools() {
		toolName := name
		toolCopy := *tool
		mcp.AddTool(server, &toolCopy, func(ctx context.Context, callReq *mcp.CallToolRequest, args map[string]any) (*mcp.CallToolResult, any, error) {
			return g.handleToolCall(ctx, agent, toolName, callReq, args)
		})
	}
	return server
}

func (g *Gateway) filteredListTools(ctx context.Context, agent *models.Agent, next mcp.MethodHandler, method string, req mcp.Request) (mcp.Result, error) {
	var names []string
	for name := range g.downstream.Tools() {
		names = append(names, name)
	}
	allowed := g.policy.FilterToolsBySkill(agent, names)

	result, err := next(ctx, method, req)
	if err != nil {
		return nil, err
	}
	listResult, ok := result.(*mcp.ListToolsResult)
	if !ok {
		return result, nil
	}
	filtered := make([]*mcp.Tool, 0, len(allowed))
	allowedSet := make(map[string]struct{}, len(allowed))
	for _, n := range allowed {
		allowedSet[n] = struct{}{}
	}
	for _, tool := range listResult.Tools {
		if _, ok := allowedSet[tool.Name]; ok {
			filtered = append(filtered, tool)
		}
	}
	listResult.Tools = filtered
	return listResult, nil
}

func (g *Gateway) handleToolCall(ctx context.Context, agent *models.Agent, toolName string, callReq *mcp.CallToolRequest, args map[string]any) (*mcp.CallToolResult, any, error) {
	start := time.Now()
	policies, err := g.loadPolicies()
	if err != nil {
		return g.deniedResult(toolName, "policy_load_error", start, agent, args), nil, nil
	}

	decision := g.policy.Enforce(agent, policies, toolName, args)
	if !decision.Allowed {
		return g.deniedResult(toolName, decision.Reason, start, agent, args), nil, nil
	}

	result, err := g.downstream.Call(ctx, toolName, args)
	latency := time.Since(start).Milliseconds()
	if err != nil {
		g.writeAudit(agent, toolName, "call", args, "error", err.Error(), latency)
		return &mcp.CallToolResult{
			IsError: true,
			Content: []mcp.Content{&mcp.TextContent{Text: err.Error()}},
		}, nil, nil
	}

	g.writeAudit(agent, toolName, "call", args, "allowed", "", latency)
	return result, result.StructuredContent, nil
}

func (g *Gateway) deniedResult(toolName, reason string, start time.Time, agent *models.Agent, args map[string]any) *mcp.CallToolResult {
	latency := time.Since(start).Milliseconds()
	g.writeAudit(agent, toolName, "call", args, "denied", reason, latency)
	msg := fmt.Sprintf("policy denied: %s", reason)
	return &mcp.CallToolResult{
		IsError: true,
		Content: []mcp.Content{&mcp.TextContent{Text: msg}},
	}
}

func (g *Gateway) writeAudit(agent *models.Agent, toolName, action string, args map[string]any, outcome, reason string, latency int64) {
	var agentID, userID *uuid.UUID
	if agent != nil {
		agentID = &agent.ID
		userID = &agent.OwnerUserID
	}
	entry := models.AuditLog{
		AgentID:         agentID,
		UserID:          userID,
		ToolName:        toolName,
		Action:          action,
		SanitizedParams: audit.SanitizeParams(args),
		Outcome:         outcome,
		Reason:          reason,
		LatencyMS:       latency,
	}
	if err := g.audit.Write(entry); err != nil {
		slog.Error("audit write failed", "error", err)
	}
}

func ParseArgs(raw json.RawMessage) map[string]any {
	if len(raw) == 0 {
		return nil
	}
	var args map[string]any
	_ = json.Unmarshal(raw, &args)
	return args
}

func ExtractBearer(h string) string {
	return strings.TrimPrefix(strings.TrimSpace(h), "Bearer ")
}

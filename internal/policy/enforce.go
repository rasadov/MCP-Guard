package policy

import (
	"encoding/json"
	"fmt"
	"slices"

	"github.com/rasadov/mcp-guard/internal/models"
)

type Decision struct {
	Allowed bool   `json:"allowed"`
	Reason  string `json:"reason,omitempty"`
}

type Engine struct{}

func NewEngine() *Engine {
	return &Engine{}
}

func (e *Engine) Enforce(agent *models.Agent, policies []models.Policy, toolName string, params map[string]any) Decision {
	if agent == nil {
		return Decision{Allowed: false, Reason: "unknown_agent"}
	}

	if agent.Skill == nil {
		return Decision{Allowed: false, Reason: "no_skill"}
	}
	skillTools, err := ParseToolList(agent.Skill.Tools)
	if err != nil {
		return Decision{Allowed: false, Reason: "invalid_skill"}
	}
	if !ToolAllowedBySkill(skillTools, toolName) {
		return Decision{Allowed: false, Reason: "skill_denied"}
	}

	for _, p := range policies {
		if !p.Enabled {
			continue
		}
		rules, err := ParseRules(p.Rules)
		if err != nil {
			return Decision{Allowed: false, Reason: "invalid_policy"}
		}
		if d := e.checkRules(agent, rules, toolName, params); !d.Allowed {
			return d
		}
	}

	return Decision{Allowed: true}
}

func (e *Engine) checkRules(agent *models.Agent, rules Rules, toolName string, params map[string]any) Decision {
	if slices.Contains(rules.DenyTools, toolName) {
		return Decision{Allowed: false, Reason: "policy_denied:tool_blocked"}
	}

	if agent.Owner.Role != "" && IsWriteTool(rules, toolName) {
		if slices.Contains(rules.DenyWriteForRoles, agent.Owner.Role) {
			return Decision{Allowed: false, Reason: "policy_denied:write_role"}
		}
	}

	if channels, ok := rules.ChannelAllowlist[toolName]; ok && len(channels) > 0 {
		channelID := extractChannelID(params)
		if channelID == "" {
			return Decision{Allowed: false, Reason: "policy_denied:missing_channel"}
		}
		if !slices.Contains(channels, channelID) {
			return Decision{Allowed: false, Reason: fmt.Sprintf("policy_denied:channel_%s", channelID)}
		}
	}

	return Decision{Allowed: true}
}

func (e *Engine) FilterTools(agent *models.Agent, policies []models.Policy, tools []string) []string {
	var allowed []string
	for _, tool := range tools {
		if e.Enforce(agent, policies, tool, nil).Allowed {
			allowed = append(allowed, tool)
		}
	}
	return allowed
}

// FilterToolsBySkill exposes skill-permitted tools in tools/list. Policy denials
// are enforced on tools/call so blocked attempts appear in the audit log.
func (e *Engine) FilterToolsBySkill(agent *models.Agent, tools []string) []string {
	if agent == nil || agent.Skill == nil {
		return nil
	}
	skillTools, err := ParseToolList(agent.Skill.Tools)
	if err != nil {
		return nil
	}
	var allowed []string
	for _, tool := range tools {
		if ToolAllowedBySkill(skillTools, tool) {
			allowed = append(allowed, tool)
		}
	}
	return allowed
}

func extractChannelID(params map[string]any) string {
	if params == nil {
		return ""
	}
	for _, key := range []string{"channel_id", "channel", "channelId"} {
		if v, ok := params[key]; ok {
			if s, ok := v.(string); ok {
				return s
			}
		}
	}
	return ""
}

func ParamsFromRaw(raw json.RawMessage) map[string]any {
	if len(raw) == 0 {
		return nil
	}
	var params map[string]any
	_ = json.Unmarshal(raw, &params)
	return params
}

package policy

import (
	"encoding/json"
	"testing"

	"github.com/google/uuid"
	"github.com/rasadov/mcp-guard/internal/models"
	"gorm.io/datatypes"
)

func TestEnforceSkillDenied(t *testing.T) {
	engine := NewEngine()
	tools, _ := json.Marshal([]string{"slack.channels_list"})
	agent := &models.Agent{
		Skill: &models.Skill{Tools: datatypes.JSON(tools)},
	}
	decision := engine.Enforce(agent, nil, "slack.conversations_add_message", nil)
	if decision.Allowed {
		t.Fatal("expected skill_denied")
	}
}

func TestEnforceDenyTool(t *testing.T) {
	engine := NewEngine()
	tools, _ := json.Marshal([]string{"slack.usergroups_create"})
	rules, _ := json.Marshal(Rules{DenyTools: []string{"slack.usergroups_create"}})
	agent := &models.Agent{
		Owner: models.User{Role: "admin"},
		Skill: &models.Skill{Tools: datatypes.JSON(tools)},
	}
	policies := []models.Policy{{Enabled: true, Rules: datatypes.JSON(rules)}}
	decision := engine.Enforce(agent, policies, "slack.usergroups_create", nil)
	if decision.Allowed {
		t.Fatal("expected policy deny")
	}
}

func TestEnforceWriteRoleDenied(t *testing.T) {
	engine := NewEngine()
	tools, _ := json.Marshal([]string{"slack.conversations_add_message"})
	rules, _ := json.Marshal(Rules{
		WriteTools:        []string{"slack.conversations_add_message"},
		DenyWriteForRoles: []string{"user"},
	})
	agent := &models.Agent{
		Owner: models.User{Role: "user"},
		Skill: &models.Skill{Tools: datatypes.JSON(tools)},
	}
	policies := []models.Policy{{Enabled: true, Rules: datatypes.JSON(rules)}}
	decision := engine.Enforce(agent, policies, "slack.conversations_add_message", map[string]any{"channel_id": "C00000000"})
	if decision.Allowed {
		t.Fatal("expected write role deny")
	}
}

func TestEnforceChannelAllowlist(t *testing.T) {
	engine := NewEngine()
	tools, _ := json.Marshal([]string{"slack.conversations_add_message"})
	rules, _ := json.Marshal(Rules{
		WriteTools: []string{"slack.conversations_add_message"},
		ChannelAllowlist: map[string][]string{
			"slack.conversations_add_message": {"C111"},
		},
	})
	agent := &models.Agent{
		Owner: models.User{Role: "admin"},
		Skill: &models.Skill{Tools: datatypes.JSON(tools)},
	}
	policies := []models.Policy{{Enabled: true, Rules: datatypes.JSON(rules)}}
	decision := engine.Enforce(agent, policies, "slack.conversations_add_message", map[string]any{"channel_id": "C222"})
	if decision.Allowed {
		t.Fatal("expected channel deny")
	}
}

func TestFilterTools(t *testing.T) {
	engine := NewEngine()
	tools, _ := json.Marshal([]string{"slack.channels_list", "slack.conversations_add_message"})
	rules, _ := json.Marshal(Rules{DenyTools: []string{"slack.conversations_add_message"}})
	agent := &models.Agent{
		ID:    uuid.New(),
		Owner: models.User{Role: "admin"},
		Skill: &models.Skill{Tools: datatypes.JSON(tools)},
	}
	policies := []models.Policy{{Enabled: true, Rules: datatypes.JSON(rules)}}
	filtered := engine.FilterTools(agent, policies, []string{"slack.channels_list", "slack.conversations_add_message"})
	if len(filtered) != 1 || filtered[0] != "slack.channels_list" {
		t.Fatalf("unexpected filtered tools: %#v", filtered)
	}
}

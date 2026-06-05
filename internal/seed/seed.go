package seed

import (
	"encoding/json"
	"log/slog"

	"github.com/google/uuid"
	"github.com/rasadov/mcp-guard/internal/models"
	"golang.org/x/crypto/bcrypt"
	"gorm.io/datatypes"
	"gorm.io/gorm"
)

const DemoAPIKey = "mcpg_demo_7f3a9b2c1d4e5f6a8b9c0d1e2f3a4b5c"

func Run(db *gorm.DB) error {
	var count int64
	if err := db.Model(&models.User{}).Count(&count).Error; err != nil {
		return err
	}
	if count > 0 {
		slog.Info("seed skipped, data already exists")
		return nil
	}

	adminID := uuid.New()
	userID := uuid.New()
	readonlySkillID := uuid.New()
	posterSkillID := uuid.New()
	agentID := uuid.New()

	readonlyTools, _ := json.Marshal([]string{
		"slack.conversations_history",
		"slack.conversations_replies",
		"slack.conversations_search_messages",
		"slack.channels_list",
	})
	posterTools, _ := json.Marshal([]string{
		"slack.conversations_history",
		"slack.conversations_replies",
		"slack.conversations_search_messages",
		"slack.channels_list",
		"slack.conversations_add_message",
	})
	posterConstraints, _ := json.Marshal(map[string]any{
		"allowed_channels": []string{"C00000000"},
	})

	policyRules, _ := json.Marshal(map[string]any{
		"deny_tools": []string{
			"slack.usergroups_create",
			"slack.usergroups_update",
			"slack.usergroups_users_update",
			"slack.reactions_add",
			"slack.reactions_remove",
		},
		"deny_write_for_roles": []string{"user"},
		"write_tools": []string{
			"slack.conversations_add_message",
			"slack.reactions_add",
			"slack.reactions_remove",
		},
		"channel_allowlist": map[string][]string{
			"slack.conversations_add_message": {"C00000000"},
		},
	})

	connectorArgs, _ := json.Marshal([]string{})
	connectorEnv, _ := json.Marshal(map[string]string{
		"SLACK_MCP_XOXP_TOKEN": "${SLACK_BOT_TOKEN}",
	})

	hash, err := bcrypt.GenerateFromPassword([]byte(DemoAPIKey), bcrypt.DefaultCost)
	if err != nil {
		return err
	}

	records := []any{
		&models.User{ID: adminID, Email: "admin@mcpguard.local", Name: "Admin", Role: "admin", Team: "platform"},
		&models.User{ID: userID, Email: "user@mcpguard.local", Name: "User", Role: "user", Team: "marketing"},
		&models.Skill{ID: readonlySkillID, Name: "Marketing Readonly", Slug: "marketing-readonly", Description: "Read-only Slack access", Tools: datatypes.JSON(readonlyTools)},
		&models.Skill{ID: posterSkillID, Name: "Marketing Poster", Slug: "marketing-poster", Description: "Read + post to allowed channel", Tools: datatypes.JSON(posterTools), Constraints: datatypes.JSON(posterConstraints)},
		&models.Policy{Name: "default", Description: "Default governance policy", Rules: datatypes.JSON(policyRules), Enabled: true},
		&models.Agent{ID: agentID, Name: "gemini-demo", OwnerUserID: adminID, SkillID: &readonlySkillID},
		&models.APIKey{AgentID: agentID, Prefix: "demo", Hash: string(hash)},
		&models.Connector{Name: "Slack", Slug: "slack", Command: "slack-mcp-server", Args: datatypes.JSON(connectorArgs), Env: datatypes.JSON(connectorEnv), Enabled: true},
	}

	for _, record := range records {
		if err := db.Create(record).Error; err != nil {
			return err
		}
	}

	slog.Info("seed data created", "demo_api_key", DemoAPIKey)
	return nil
}

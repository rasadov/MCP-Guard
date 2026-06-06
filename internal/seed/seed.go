package seed

import (
	"encoding/json"
	"log/slog"

	"github.com/google/uuid"
	"github.com/rasadov/mcp-guard/internal/models"
	"gorm.io/datatypes"
	"gorm.io/gorm"
)

func Run(db *gorm.DB) error {
	if err := cleanupDemoCredentials(db); err != nil {
		return err
	}

	var count int64
	if err := db.Model(&models.User{}).Count(&count).Error; err != nil {
		return err
	}
	if count == 0 {
		if err := seedAll(db); err != nil {
			return err
		}
	}
	return ensureDefaultPolicy(db)
}

func cleanupDemoCredentials(db *gorm.DB) error {
	if err := db.Where("prefix = ?", "demo").Delete(&models.APIKey{}).Error; err != nil {
		return err
	}
	result := db.Where("name = ?", "gemini-demo").Delete(&models.Agent{})
	if result.Error != nil {
		return result.Error
	}
	if result.RowsAffected > 0 {
		slog.Info("removed demo agent credentials")
	}
	return nil
}

func defaultPolicyRules() datatypes.JSON {
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
	return datatypes.JSON(policyRules)
}

func ensureDefaultPolicy(db *gorm.DB) error {
	var count int64
	if err := db.Model(&models.Policy{}).Count(&count).Error; err != nil {
		return err
	}
	if count > 0 {
		return nil
	}
	policy := models.Policy{
		Name:        "default",
		Description: "Default governance policy",
		Rules:       defaultPolicyRules(),
		Enabled:     true,
	}
	if err := db.Create(&policy).Error; err != nil {
		return err
	}
	slog.Info("default policy restored")
	return nil
}

func seedAll(db *gorm.DB) error {
	adminID := uuid.New()
	userID := uuid.New()
	readonlySkillID := uuid.New()
	posterSkillID := uuid.New()

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

	policyRules := defaultPolicyRules()

	connectorArgs, _ := json.Marshal([]string{})
	connectorEnv, _ := json.Marshal(map[string]string{
		"SLACK_MCP_XOXP_TOKEN": "${SLACK_BOT_TOKEN}",
	})

	records := []any{
		&models.User{ID: adminID, Email: "admin@mcpguard.local", Name: "Admin", Role: "admin", Team: "platform"},
		&models.User{ID: userID, Email: "user@mcpguard.local", Name: "User", Role: "user", Team: "marketing"},
		&models.Skill{ID: readonlySkillID, Name: "Marketing Readonly", Slug: "marketing-readonly", Description: "Read-only Slack access", Tools: datatypes.JSON(readonlyTools)},
		&models.Skill{ID: posterSkillID, Name: "Marketing Poster", Slug: "marketing-poster", Description: "Read + post to allowed channel", Tools: datatypes.JSON(posterTools), Constraints: datatypes.JSON(posterConstraints)},
		&models.Policy{Name: "default", Description: "Default governance policy", Rules: policyRules, Enabled: true},
		&models.Connector{Name: "Slack", Slug: "slack", Command: "slack-mcp-server", Args: datatypes.JSON(connectorArgs), Env: datatypes.JSON(connectorEnv), Enabled: true},
	}

	return db.Transaction(func(tx *gorm.DB) error {
		for _, record := range records {
			if err := tx.Create(record).Error; err != nil {
				return err
			}
		}
		slog.Info("seed data created")
		return nil
	})
}

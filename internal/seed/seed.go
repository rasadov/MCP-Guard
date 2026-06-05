package seed

import (
	"encoding/json"
	"errors"
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
	if count == 0 {
		if err := seedAll(db); err != nil {
			return err
		}
	} else {
		slog.Info("seed skipped full bootstrap, ensuring demo credentials")
	}
	if err := ensureDefaultPolicy(db); err != nil {
		return err
	}
	return ensureDemoCredentials(db)
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

	policyRules := defaultPolicyRules()

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
		&models.Policy{Name: "default", Description: "Default governance policy", Rules: policyRules, Enabled: true},
		&models.Agent{ID: agentID, Name: "gemini-demo", OwnerUserID: adminID, SkillID: &readonlySkillID},
		&models.APIKey{AgentID: agentID, Prefix: "demo", Hash: string(hash)},
		&models.Connector{Name: "Slack", Slug: "slack", Command: "slack-mcp-server", Args: datatypes.JSON(connectorArgs), Env: datatypes.JSON(connectorEnv), Enabled: true},
	}

	return db.Transaction(func(tx *gorm.DB) error {
		for _, record := range records {
			if err := tx.Create(record).Error; err != nil {
				return err
			}
		}
		slog.Info("seed data created", "demo_api_key", DemoAPIKey)
		return nil
	})
}

func ensureDemoCredentials(db *gorm.DB) error {
	hash, err := bcrypt.GenerateFromPassword([]byte(DemoAPIKey), bcrypt.DefaultCost)
	if err != nil {
		return err
	}

	var existing models.APIKey
	err = db.Where("prefix = ?", "demo").First(&existing).Error
	if err == nil {
		if bcrypt.CompareHashAndPassword([]byte(existing.Hash), []byte(DemoAPIKey)) == nil {
			return nil
		}
		if err := db.Model(&existing).Update("hash", string(hash)).Error; err != nil {
			return err
		}
		slog.Info("demo api key hash refreshed")
		return nil
	}
	if !errors.Is(err, gorm.ErrRecordNotFound) {
		return err
	}

	readonlyTools, _ := json.Marshal([]string{
		"slack.conversations_history",
		"slack.conversations_replies",
		"slack.conversations_search_messages",
		"slack.channels_list",
	})

	var admin models.User
	if err := db.Where("email = ?", "admin@mcpguard.local").First(&admin).Error; err != nil {
		if !errors.Is(err, gorm.ErrRecordNotFound) {
			return err
		}
		admin = models.User{Email: "admin@mcpguard.local", Name: "Admin", Role: "admin", Team: "platform"}
		if err := db.Create(&admin).Error; err != nil {
			return err
		}
	}

	var skill models.Skill
	if err := db.Where("slug = ?", "marketing-readonly").First(&skill).Error; err != nil {
		if !errors.Is(err, gorm.ErrRecordNotFound) {
			return err
		}
		skill = models.Skill{
			Name:        "Marketing Readonly",
			Slug:        "marketing-readonly",
			Description: "Read-only Slack access",
			Tools:       datatypes.JSON(readonlyTools),
		}
		if err := db.Create(&skill).Error; err != nil {
			return err
		}
	}

	var agent models.Agent
	if err := db.Where("name = ?", "gemini-demo").First(&agent).Error; err != nil {
		if !errors.Is(err, gorm.ErrRecordNotFound) {
			return err
		}
		skillID := skill.ID
		agent = models.Agent{Name: "gemini-demo", OwnerUserID: admin.ID, SkillID: &skillID}
		if err := db.Create(&agent).Error; err != nil {
			return err
		}
	}

	key := models.APIKey{AgentID: agent.ID, Prefix: "demo", Hash: string(hash)}
	if err := db.Create(&key).Error; err != nil {
		return err
	}
	slog.Info("demo credentials restored", "demo_api_key", DemoAPIKey)
	return nil
}

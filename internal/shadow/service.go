package shadow

import (
	"time"

	"github.com/rasadov/mcp-guard/internal/models"
	"gorm.io/gorm"
)

type Flag struct {
	AgentName string    `json:"agent_name"`
	ToolName  string    `json:"tool_name"`
	Source    string    `json:"source"`
	Message   string    `json:"message"`
	Detected  time.Time `json:"detected"`
}

type Service struct {
	db *gorm.DB
}

func NewService(db *gorm.DB) *Service {
	return &Service{db: db}
}

func (s *Service) Record(event models.ShadowEvent) error {
	return s.db.Create(&event).Error
}

func (s *Service) Detect() ([]Flag, error) {
	var events []models.ShadowEvent
	if err := s.db.Order("created_at desc").Limit(200).Find(&events).Error; err != nil {
		return nil, err
	}

	var flags []Flag
	for _, event := range events {
		var auditCount int64
		s.db.Model(&models.AuditLog{}).
			Joins("JOIN agents ON agents.id = audit_logs.agent_id").
			Where("agents.name = ? AND audit_logs.tool_name = ? AND audit_logs.outcome = ?", event.AgentName, event.ToolName, "allowed").
			Count(&auditCount)
		if auditCount == 0 {
			flags = append(flags, Flag{
				AgentName: event.AgentName,
				ToolName:  event.ToolName,
				Source:    event.Source,
				Message:   "Direct tool call detected outside MCP Guard gateway",
				Detected:  event.CreatedAt,
			})
		}
	}
	return flags, nil
}

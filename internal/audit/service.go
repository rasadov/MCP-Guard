package audit

import (
	"encoding/csv"
	"encoding/json"
	"io"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/rasadov/mcp-guard/internal/models"
	"gorm.io/datatypes"
	"gorm.io/gorm"
)

type Service struct {
	db *gorm.DB
}

func NewService(db *gorm.DB) *Service {
	return &Service{db: db}
}

type Query struct {
	From    *time.Time
	To      *time.Time
	AgentID *uuid.UUID
	Limit   int
}

func (s *Service) Write(entry models.AuditLog) error {
	return s.db.Create(&entry).Error
}

func (s *Service) List(q Query) ([]models.AuditLog, error) {
	limit := q.Limit
	if limit <= 0 {
		limit = 100
	}
	query := s.db.Order("created_at desc").Limit(limit)
	if q.From != nil {
		query = query.Where("created_at >= ?", *q.From)
	}
	if q.To != nil {
		query = query.Where("created_at <= ?", *q.To)
	}
	if q.AgentID != nil {
		query = query.Where("agent_id = ?", *q.AgentID)
	}
	var logs []models.AuditLog
	return logs, query.Find(&logs).Error
}

func SanitizeParams(params map[string]any) datatypes.JSON {
	if params == nil {
		return nil
	}
	safe := make(map[string]any, len(params))
	for k, v := range params {
		lower := strings.ToLower(k)
		if strings.Contains(lower, "token") || strings.Contains(lower, "secret") || strings.Contains(lower, "password") {
			safe[k] = "[REDACTED]"
			continue
		}
		if s, ok := v.(string); ok && len(s) > 500 {
			safe[k] = s[:500] + "..."
			continue
		}
		safe[k] = v
	}
	b, _ := json.Marshal(safe)
	return datatypes.JSON(b)
}

func (s *Service) ExportJSON(w io.Writer, q Query) error {
	logs, err := s.List(q)
	if err != nil {
		return err
	}
	enc := json.NewEncoder(w)
	enc.SetIndent("", "  ")
	return enc.Encode(logs)
}

func (s *Service) ExportCSV(w io.Writer, q Query) error {
	logs, err := s.List(q)
	if err != nil {
		return err
	}
	writer := csv.NewWriter(w)
	_ = writer.Write([]string{"id", "agent_id", "user_id", "tool_name", "action", "outcome", "reason", "latency_ms", "created_at"})
	for _, log := range logs {
		agentID := ""
		if log.AgentID != nil {
			agentID = log.AgentID.String()
		}
		userID := ""
		if log.UserID != nil {
			userID = log.UserID.String()
		}
		_ = writer.Write([]string{
			log.ID.String(),
			agentID,
			userID,
			log.ToolName,
			log.Action,
			log.Outcome,
			log.Reason,
			formatInt(log.LatencyMS),
			log.CreatedAt.Format(time.RFC3339),
		})
	}
	writer.Flush()
	return writer.Error()
}

func formatInt(v int64) string {
	b, _ := json.Marshal(v)
	return strings.Trim(string(b), "\"")
}

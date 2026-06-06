package api

import (
	"slices"

	"github.com/google/uuid"
	"github.com/rasadov/mcp-guard/internal/auth"
	"github.com/rasadov/mcp-guard/internal/models"
	"gorm.io/gorm"
)

func (h *Handlers) ownedAgentIDs(claims *auth.Claims) (scoped bool, ids []uuid.UUID, err error) {
	if claims.Role == "admin" {
		return false, nil, nil
	}
	var agents []models.Agent
	if err := h.db.Where("owner_user_id = ?", claims.UserID).Select("id").Find(&agents).Error; err != nil {
		return true, nil, err
	}
	ids = make([]uuid.UUID, len(agents))
	for i, agent := range agents {
		ids[i] = agent.ID
	}
	return true, ids, nil
}

func applyAgentIDScope(db *gorm.DB, scoped bool, agentIDs []uuid.UUID) *gorm.DB {
	if !scoped {
		return db
	}
	if len(agentIDs) == 0 {
		return db.Where("1 = 0")
	}
	return db.Where("agent_id IN ?", agentIDs)
}

func agentAllowed(scoped bool, agentIDs []uuid.UUID, agentID uuid.UUID) bool {
	if !scoped {
		return true
	}
	return slices.Contains(agentIDs, agentID)
}

package models

import (
	"time"

	"github.com/google/uuid"
	"gorm.io/datatypes"
	"gorm.io/gorm"
)

type User struct {
	ID        uuid.UUID `gorm:"type:uuid;primaryKey" json:"id"`
	GoogleSub *string   `gorm:"uniqueIndex" json:"google_sub,omitempty"`
	Email     string    `gorm:"uniqueIndex;not null" json:"email"`
	Name      string    `json:"name"`
	Role      string    `gorm:"not null;default:user" json:"role"`
	Team      string    `json:"team,omitempty"`
	CreatedAt time.Time `json:"created_at"`
	UpdatedAt time.Time `json:"updated_at"`
}

type Skill struct {
	ID          uuid.UUID      `gorm:"type:uuid;primaryKey" json:"id"`
	Name        string         `gorm:"uniqueIndex;not null" json:"name"`
	Slug        string         `gorm:"uniqueIndex;not null" json:"slug"`
	Description string         `json:"description"`
	Tools       datatypes.JSON `gorm:"type:jsonb;not null" json:"tools"`
	Constraints datatypes.JSON `gorm:"type:jsonb" json:"constraints,omitempty"`
	CreatedAt   time.Time      `json:"created_at"`
	UpdatedAt   time.Time      `json:"updated_at"`
}

type Policy struct {
	ID          uuid.UUID      `gorm:"type:uuid;primaryKey" json:"id"`
	Name        string         `gorm:"uniqueIndex;not null" json:"name"`
	Description string         `json:"description"`
	Rules       datatypes.JSON `gorm:"type:jsonb;not null" json:"rules"`
	Enabled     bool           `gorm:"not null;default:true" json:"enabled"`
	CreatedAt   time.Time      `json:"created_at"`
	UpdatedAt   time.Time      `json:"updated_at"`
}

type Agent struct {
	ID          uuid.UUID  `gorm:"type:uuid;primaryKey" json:"id"`
	Name        string     `gorm:"not null" json:"name"`
	OwnerUserID uuid.UUID  `gorm:"type:uuid;not null" json:"owner_user_id"`
	Owner       User       `gorm:"foreignKey:OwnerUserID" json:"owner,omitempty"`
	SkillID     *uuid.UUID `gorm:"type:uuid" json:"skill_id,omitempty"`
	Skill       *Skill     `gorm:"foreignKey:SkillID" json:"skill,omitempty"`
	CreatedAt   time.Time  `json:"created_at"`
	UpdatedAt   time.Time  `json:"updated_at"`
}

type APIKey struct {
	ID        uuid.UUID `gorm:"type:uuid;primaryKey" json:"id"`
	AgentID   uuid.UUID `gorm:"type:uuid;not null;index" json:"agent_id"`
	Agent     Agent     `gorm:"foreignKey:AgentID" json:"agent,omitempty"`
	Prefix    string    `gorm:"uniqueIndex;not null" json:"prefix"`
	Hash      string    `gorm:"not null" json:"-"`
	CreatedAt time.Time `json:"created_at"`
}

type Connector struct {
	ID        uuid.UUID      `gorm:"type:uuid;primaryKey" json:"id"`
	Name      string         `gorm:"uniqueIndex;not null" json:"name"`
	Slug      string         `gorm:"uniqueIndex;not null" json:"slug"`
	Command   string         `gorm:"not null" json:"command"`
	Args      datatypes.JSON `gorm:"type:jsonb" json:"args"`
	Env       datatypes.JSON `gorm:"type:jsonb" json:"env"`
	Enabled   bool           `gorm:"not null;default:true" json:"enabled"`
	CreatedAt time.Time      `json:"created_at"`
	UpdatedAt time.Time      `json:"updated_at"`
}

type AuditLog struct {
	ID              uuid.UUID      `gorm:"type:uuid;primaryKey" json:"id"`
	AgentID         *uuid.UUID     `gorm:"type:uuid;index" json:"agent_id,omitempty"`
	UserID          *uuid.UUID     `gorm:"type:uuid;index" json:"user_id,omitempty"`
	ToolName        string         `gorm:"not null;index" json:"tool_name"`
	Action          string         `gorm:"not null" json:"action"`
	SanitizedParams datatypes.JSON `gorm:"type:jsonb" json:"sanitized_params,omitempty"`
	Outcome         string         `gorm:"not null;index" json:"outcome"`
	Reason          string         `json:"reason,omitempty"`
	LatencyMS       int64          `json:"latency_ms"`
	CreatedAt       time.Time      `gorm:"index" json:"created_at"`
}

type ShadowEvent struct {
	ID        uuid.UUID      `gorm:"type:uuid;primaryKey" json:"id"`
	AgentName string         `gorm:"not null;index" json:"agent_name"`
	ToolName  string         `gorm:"not null" json:"tool_name"`
	Source    string         `gorm:"not null" json:"source"`
	Metadata  datatypes.JSON `gorm:"type:jsonb" json:"metadata,omitempty"`
	CreatedAt time.Time      `gorm:"index" json:"created_at"`
}

type Session struct {
	ID        uuid.UUID `gorm:"type:uuid;primaryKey" json:"id"`
	AgentID   uuid.UUID `gorm:"type:uuid;not null;index" json:"agent_id"`
	Agent     Agent     `gorm:"foreignKey:AgentID" json:"agent,omitempty"`
	LastSeen  time.Time `gorm:"index" json:"last_seen"`
	CreatedAt time.Time `json:"created_at"`
}

func AutoMigrate(db *gorm.DB) error {
	return db.AutoMigrate(
		&User{},
		&Skill{},
		&Policy{},
		&Agent{},
		&APIKey{},
		&Connector{},
		&AuditLog{},
		&ShadowEvent{},
		&Session{},
	)
}

func (u *User) BeforeCreate(tx *gorm.DB) error {
	if u.ID == uuid.Nil {
		u.ID = uuid.New()
	}
	return nil
}

func (s *Skill) BeforeCreate(tx *gorm.DB) error {
	if s.ID == uuid.Nil {
		s.ID = uuid.New()
	}
	return nil
}

func (p *Policy) BeforeCreate(tx *gorm.DB) error {
	if p.ID == uuid.Nil {
		p.ID = uuid.New()
	}
	return nil
}

func (a *Agent) BeforeCreate(tx *gorm.DB) error {
	if a.ID == uuid.Nil {
		a.ID = uuid.New()
	}
	return nil
}

func (k *APIKey) BeforeCreate(tx *gorm.DB) error {
	if k.ID == uuid.Nil {
		k.ID = uuid.New()
	}
	return nil
}

func (c *Connector) BeforeCreate(tx *gorm.DB) error {
	if c.ID == uuid.Nil {
		c.ID = uuid.New()
	}
	return nil
}

func (a *AuditLog) BeforeCreate(tx *gorm.DB) error {
	if a.ID == uuid.Nil {
		a.ID = uuid.New()
	}
	return nil
}

func (s *ShadowEvent) BeforeCreate(tx *gorm.DB) error {
	if s.ID == uuid.Nil {
		s.ID = uuid.New()
	}
	return nil
}

func (s *Session) BeforeCreate(tx *gorm.DB) error {
	if s.ID == uuid.Nil {
		s.ID = uuid.New()
	}
	return nil
}

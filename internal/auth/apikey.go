package auth

import (
	"crypto/rand"
	"encoding/hex"
	"errors"
	"strings"

	"github.com/google/uuid"
	"github.com/rasadov/mcp-guard/internal/models"
	"golang.org/x/crypto/bcrypt"
	"gorm.io/gorm"
)

var ErrInvalidAPIKey = errors.New("invalid api key")

type APIKeyService struct {
	db *gorm.DB
}

func NewAPIKeyService(db *gorm.DB) *APIKeyService {
	return &APIKeyService{db: db}
}

func (s *APIKeyService) Validate(raw string) (*models.Agent, error) {
	raw = strings.TrimSpace(raw)
	raw = strings.TrimPrefix(raw, "Bearer ")
	if !strings.HasPrefix(raw, "mcpg_") {
		return nil, ErrInvalidAPIKey
	}
	parts := strings.SplitN(raw, "_", 3)
	if len(parts) != 3 {
		return nil, ErrInvalidAPIKey
	}
	prefix := parts[1]

	var key models.APIKey
	if err := s.db.Where("prefix = ?", prefix).First(&key).Error; err != nil {
		return nil, ErrInvalidAPIKey
	}
	if err := bcrypt.CompareHashAndPassword([]byte(key.Hash), []byte(raw)); err != nil {
		return nil, ErrInvalidAPIKey
	}

	var agent models.Agent
	if err := s.db.Preload("Owner").Preload("Skill").First(&agent, "id = ?", key.AgentID).Error; err != nil {
		return nil, ErrInvalidAPIKey
	}
	return &agent, nil
}

func (s *APIKeyService) Create(agentID uuid.UUID) (string, error) {
	prefixBytes := make([]byte, 4)
	if _, err := rand.Read(prefixBytes); err != nil {
		return "", err
	}
	prefix := hex.EncodeToString(prefixBytes)
	secretBytes := make([]byte, 16)
	if _, err := rand.Read(secretBytes); err != nil {
		return "", err
	}
	secret := hex.EncodeToString(secretBytes)
	raw := "mcpg_" + prefix + "_" + secret

	hash, err := bcrypt.GenerateFromPassword([]byte(raw), bcrypt.DefaultCost)
	if err != nil {
		return "", err
	}

	key := models.APIKey{AgentID: agentID, Prefix: prefix, Hash: string(hash)}
	if err := s.db.Create(&key).Error; err != nil {
		return "", err
	}
	return raw, nil
}

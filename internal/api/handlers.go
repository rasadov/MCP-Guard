package api

import (
	"encoding/json"
	"net/http"
	"slices"
	"strconv"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/rasadov/mcp-guard/internal/audit"
	"github.com/rasadov/mcp-guard/internal/auth"
	"github.com/rasadov/mcp-guard/internal/config"
	"github.com/rasadov/mcp-guard/internal/models"
	"github.com/rasadov/mcp-guard/internal/policy"
	"github.com/rasadov/mcp-guard/internal/shadow"
	"gorm.io/datatypes"
	"gorm.io/gorm"
)

type Handlers struct {
	db       *gorm.DB
	cfg      config.Config
	jwt      *auth.JWTService
	google   *auth.GoogleAuth
	apiKeys  *auth.APIKeyService
	audit    *audit.Service
	shadow   *shadow.Service
	policy   *policy.Engine
	toolList func() []string
}

func NewHandlers(
	db *gorm.DB,
	cfg config.Config,
	jwt *auth.JWTService,
	google *auth.GoogleAuth,
	apiKeys *auth.APIKeyService,
	auditSvc *audit.Service,
	shadowSvc *shadow.Service,
	policyEngine *policy.Engine,
	toolList func() []string,
) *Handlers {
	return &Handlers{
		db:       db,
		cfg:      cfg,
		jwt:      jwt,
		google:   google,
		apiKeys:  apiKeys,
		audit:    auditSvc,
		shadow:   shadowSvc,
		policy:   policyEngine,
		toolList: toolList,
	}
}

func (h *Handlers) Register(r *gin.RouterGroup) {
	r.GET("/me", JWTMiddleware(h.jwt), h.me)

	r.GET("/audit", JWTMiddleware(h.jwt), h.listAudit)
	r.GET("/audit/export", JWTMiddleware(h.jwt), h.exportAudit)

	r.GET("/shadow", JWTMiddleware(h.jwt), RequireAdmin(), h.listShadow)
	r.POST("/shadow-events", JWTMiddleware(h.jwt), RequireAdmin(), h.createShadowEvent)

	r.GET("/tools", JWTMiddleware(h.jwt), h.listTools)
	r.GET("/stats", JWTMiddleware(h.jwt), h.stats)
	r.GET("/agents/active", JWTMiddleware(h.jwt), h.activeAgents)

	r.GET("/skills", JWTMiddleware(h.jwt), h.listSkills)
	r.POST("/skills", JWTMiddleware(h.jwt), RequireAdmin(), h.createSkill)
	r.PUT("/skills/:id", JWTMiddleware(h.jwt), RequireAdmin(), h.updateSkill)
	r.DELETE("/skills/:id", JWTMiddleware(h.jwt), RequireAdmin(), h.deleteSkill)

	r.GET("/policies", JWTMiddleware(h.jwt), RequireAdmin(), h.listPolicies)
	r.POST("/policies", JWTMiddleware(h.jwt), RequireAdmin(), h.createPolicy)
	r.PUT("/policies/:id", JWTMiddleware(h.jwt), RequireAdmin(), h.updatePolicy)
	r.PATCH("/policies/:id/deny-tools", JWTMiddleware(h.jwt), RequireAdmin(), h.patchPolicyDenyTools)
	r.DELETE("/policies/:id", JWTMiddleware(h.jwt), RequireAdmin(), h.deletePolicy)

	r.GET("/agents", JWTMiddleware(h.jwt), h.listAgents)
	r.POST("/agents", JWTMiddleware(h.jwt), h.createAgent)
	r.PUT("/agents/:id", JWTMiddleware(h.jwt), RequireAdmin(), h.updateAgent)
	r.POST("/agents/:id/rotate-key", JWTMiddleware(h.jwt), h.rotateAgentKey)
}

func (h *Handlers) RegisterAuth(r *gin.RouterGroup) {
	r.GET("/config", h.authConfig)
	r.GET("/logout", h.logout)
	r.GET("/google", h.googleLogin)
	r.GET("/google/callback", h.googleCallback)
	if h.cfg.AuthDevMode {
		r.GET("/dev-login", h.devLogin)
	}
}

func (h *Handlers) authConfig(c *gin.Context) {
	c.JSON(http.StatusOK, gin.H{
		"google_enabled":     h.google.Enabled(),
		"dev_login_enabled":  h.cfg.AuthDevMode,
	})
}

func (h *Handlers) logout(c *gin.Context) {
	c.SetSameSite(http.SameSiteLaxMode)
	c.SetCookie("mcp_guard_token", "", -1, "/", "", false, true)
	c.Redirect(http.StatusTemporaryRedirect, "/login")
}

func (h *Handlers) me(c *gin.Context) {
	claims := GetClaims(c)
	var user models.User
	if err := h.db.First(&user, "id = ?", claims.UserID).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "user not found"})
		return
	}
	c.JSON(http.StatusOK, user)
}

func (h *Handlers) googleLogin(c *gin.Context) {
	if !h.google.Enabled() {
		c.JSON(http.StatusBadRequest, gin.H{"error": "google oauth not configured"})
		return
	}
	c.Redirect(http.StatusTemporaryRedirect, h.google.AuthCodeURL("state"))
}

func (h *Handlers) googleCallback(c *gin.Context) {
	code := c.Query("code")
	if code == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "missing code"})
		return
	}
	gUser, err := h.google.ExchangeUser(c.Request.Context(), code)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	user, err := h.upsertGoogleUser(gUser)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	token, err := h.jwt.Issue(*user)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.SetSameSite(http.SameSiteLaxMode)
	c.SetCookie("mcp_guard_token", token, int(h.cfg.JWTExpiry.Seconds()), "/", "", false, true)
	c.Redirect(http.StatusTemporaryRedirect, "/")
}

func (h *Handlers) devLogin(c *gin.Context) {
	email := c.Query("email")
	if email == "" {
		email = "admin@mcpguard.local"
	}
	var user models.User
	if err := h.db.Where("email = ?", email).First(&user).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "user not found"})
		return
	}
	token, err := h.jwt.Issue(user)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.SetSameSite(http.SameSiteLaxMode)
	c.SetCookie("mcp_guard_token", token, int(h.cfg.JWTExpiry.Seconds()), "/", "", false, true)
	c.Redirect(http.StatusTemporaryRedirect, "/")
}

func (h *Handlers) upsertGoogleUser(gUser *auth.GoogleUser) (*models.User, error) {
	var user models.User
	err := h.db.Where("google_sub = ?", gUser.Sub).First(&user).Error
	if err == gorm.ErrRecordNotFound {
		sub := gUser.Sub
		user = models.User{
			GoogleSub: &sub,
			Email:     gUser.Email,
			Name:      gUser.Name,
			Role:      "user",
		}
		return &user, h.db.Create(&user).Error
	}
	if err != nil {
		return nil, err
	}
	sub := gUser.Sub
	user.GoogleSub = &sub
	user.Email = gUser.Email
	user.Name = gUser.Name
	return &user, h.db.Save(&user).Error
}

func (h *Handlers) listAudit(c *gin.Context) {
	claims := GetClaims(c)
	scoped, agentIDs, err := h.ownedAgentIDs(claims)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	q := audit.Query{Limit: queryInt(c, "limit", 100)}
	if from := c.Query("from"); from != "" {
		if t, err := time.Parse(time.RFC3339, from); err == nil {
			q.From = &t
		}
	}
	if to := c.Query("to"); to != "" {
		if t, err := time.Parse(time.RFC3339, to); err == nil {
			q.To = &t
		}
	}
	if agentID := c.Query("agent_id"); agentID != "" {
		id, err := uuid.Parse(agentID)
		if err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": "invalid agent_id"})
			return
		}
		if !agentAllowed(scoped, agentIDs, id) {
			c.JSON(http.StatusForbidden, gin.H{"error": "forbidden"})
			return
		}
		q.AgentID = &id
	} else if scoped {
		q.AgentIDs = agentIDs
	}
	logs, err := h.audit.List(q)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, logs)
}

func (h *Handlers) exportAudit(c *gin.Context) {
	claims := GetClaims(c)
	scoped, agentIDs, err := h.ownedAgentIDs(claims)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	q := audit.Query{Limit: queryInt(c, "limit", 1000)}
	if agentID := c.Query("agent_id"); agentID != "" {
		id, err := uuid.Parse(agentID)
		if err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": "invalid agent_id"})
			return
		}
		if !agentAllowed(scoped, agentIDs, id) {
			c.JSON(http.StatusForbidden, gin.H{"error": "forbidden"})
			return
		}
		q.AgentID = &id
	} else if scoped {
		q.AgentIDs = agentIDs
	}
	format := c.DefaultQuery("format", "json")
	if format == "csv" {
		c.Header("Content-Type", "text/csv")
		c.Header("Content-Disposition", "attachment; filename=audit.csv")
		if err := h.audit.ExportCSV(c.Writer, q); err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		}
		return
	}
	c.Header("Content-Type", "application/json")
	c.Header("Content-Disposition", "attachment; filename=audit.json")
	if err := h.audit.ExportJSON(c.Writer, q); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
	}
}

func (h *Handlers) listShadow(c *gin.Context) {
	flags, err := h.shadow.Detect()
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, flags)
}

func (h *Handlers) createShadowEvent(c *gin.Context) {
	var body struct {
		AgentName string         `json:"agent_name"`
		ToolName  string         `json:"tool_name"`
		Source    string         `json:"source"`
		Metadata  map[string]any `json:"metadata"`
	}
	if err := c.ShouldBindJSON(&body); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	meta, _ := json.Marshal(body.Metadata)
	event := models.ShadowEvent{
		AgentName: body.AgentName,
		ToolName:  body.ToolName,
		Source:    body.Source,
		Metadata:  datatypes.JSON(meta),
	}
	if err := h.shadow.Record(event); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusCreated, event)
}

func (h *Handlers) listTools(c *gin.Context) {
	c.JSON(http.StatusOK, gin.H{"tools": h.toolList()})
}

func (h *Handlers) stats(c *gin.Context) {
	claims := GetClaims(c)
	scoped, agentIDs, err := h.ownedAgentIDs(claims)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	var total, allowed, denied int64
	applyAgentIDScope(h.db.Model(&models.AuditLog{}), scoped, agentIDs).Count(&total)
	applyAgentIDScope(h.db.Model(&models.AuditLog{}), scoped, agentIDs).Where("outcome = ?", "allowed").Count(&allowed)
	applyAgentIDScope(h.db.Model(&models.AuditLog{}), scoped, agentIDs).Where("outcome = ?", "denied").Count(&denied)

	type toolCount struct {
		ToolName string `json:"tool_name"`
		Count    int64  `json:"count"`
	}
	var topTools []toolCount
	applyAgentIDScope(h.db.Model(&models.AuditLog{}), scoped, agentIDs).
		Select("tool_name, count(*) as count").
		Group("tool_name").
		Order("count desc").
		Limit(5).
		Scan(&topTools)

	c.JSON(http.StatusOK, gin.H{
		"total_calls":   total,
		"allowed_calls": allowed,
		"denied_calls":  denied,
		"top_tools":     topTools,
	})
}

func (h *Handlers) activeAgents(c *gin.Context) {
	claims := GetClaims(c)
	scoped, agentIDs, err := h.ownedAgentIDs(claims)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	q := h.db.Preload("Agent").Preload("Agent.Skill").Order("last_seen desc").Limit(50)
	if scoped {
		if len(agentIDs) == 0 {
			c.JSON(http.StatusOK, []models.Session{})
			return
		}
		q = q.Where("agent_id IN ?", agentIDs)
	}
	var sessions []models.Session
	q.Find(&sessions)
	c.JSON(http.StatusOK, sessions)
}

func (h *Handlers) listSkills(c *gin.Context) {
	var skills []models.Skill
	h.db.Find(&skills)
	c.JSON(http.StatusOK, skills)
}

func (h *Handlers) createSkill(c *gin.Context) {
	var body models.Skill
	if err := c.ShouldBindJSON(&body); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	if err := h.db.Create(&body).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusCreated, body)
}

func (h *Handlers) updateSkill(c *gin.Context) {
	id, err := uuid.Parse(c.Param("id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid id"})
		return
	}
	var skill models.Skill
	if err := h.db.First(&skill, "id = ?", id).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "not found"})
		return
	}
	if err := c.ShouldBindJSON(&skill); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	skill.ID = id
	if err := h.db.Save(&skill).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, skill)
}

func (h *Handlers) deleteSkill(c *gin.Context) {
	id, err := uuid.Parse(c.Param("id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid id"})
		return
	}
	if err := h.db.Delete(&models.Skill{}, "id = ?", id).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.Status(http.StatusNoContent)
}

func (h *Handlers) listPolicies(c *gin.Context) {
	var policies []models.Policy
	h.db.Find(&policies)
	c.JSON(http.StatusOK, policies)
}

func (h *Handlers) createPolicy(c *gin.Context) {
	var body models.Policy
	if err := c.ShouldBindJSON(&body); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	if err := h.db.Create(&body).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusCreated, body)
}

func (h *Handlers) updatePolicy(c *gin.Context) {
	id, err := uuid.Parse(c.Param("id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid id"})
		return
	}
	var policy models.Policy
	if err := h.db.First(&policy, "id = ?", id).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "not found"})
		return
	}
	if err := c.ShouldBindJSON(&policy); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	policy.ID = id
	if err := h.db.Save(&policy).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, policy)
}

func (h *Handlers) deletePolicy(c *gin.Context) {
	id, err := uuid.Parse(c.Param("id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid id"})
		return
	}
	if err := h.db.Delete(&models.Policy{}, "id = ?", id).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.Status(http.StatusNoContent)
}

func (h *Handlers) patchPolicyDenyTools(c *gin.Context) {
	id, err := uuid.Parse(c.Param("id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid id"})
		return
	}
	var body struct {
		ToolName string `json:"tool_name"`
		Blocked  bool   `json:"blocked"`
	}
	if err := c.ShouldBindJSON(&body); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	if body.ToolName == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "tool_name required"})
		return
	}

	var pol models.Policy
	if err := h.db.First(&pol, "id = ?", id).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "not found"})
		return
	}

	rules, err := policy.ParseRules(pol.Rules)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "invalid policy rules"})
		return
	}

	if body.Blocked {
		if !slices.Contains(rules.DenyTools, body.ToolName) {
			rules.DenyTools = append(rules.DenyTools, body.ToolName)
		}
	} else {
		rules.DenyTools = slices.DeleteFunc(rules.DenyTools, func(t string) bool {
			return t == body.ToolName
		})
	}

	encoded, err := json.Marshal(rules)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	pol.Rules = datatypes.JSON(encoded)
	if err := h.db.Save(&pol).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, pol)
}

func (h *Handlers) updateAgent(c *gin.Context) {
	id, err := uuid.Parse(c.Param("id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid id"})
		return
	}
	var body struct {
		SkillID *uuid.UUID `json:"skill_id"`
	}
	if err := c.ShouldBindJSON(&body); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	var agent models.Agent
	if err := h.db.First(&agent, "id = ?", id).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "not found"})
		return
	}
	agent.SkillID = body.SkillID
	if err := h.db.Save(&agent).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	if err := h.db.Preload("Skill").First(&agent, "id = ?", id).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, agent)
}

func (h *Handlers) listAgents(c *gin.Context) {
	claims := GetClaims(c)
	var agents []models.Agent
	q := h.db.Preload("Skill")
	if claims.Role != "admin" {
		q = q.Where("owner_user_id = ?", claims.UserID)
	}
	q.Find(&agents)
	c.JSON(http.StatusOK, agents)
}

func (h *Handlers) createAgent(c *gin.Context) {
	claims := GetClaims(c)
	var body struct {
		Name    string     `json:"name"`
		SkillID *uuid.UUID `json:"skill_id"`
	}
	if err := c.ShouldBindJSON(&body); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	agent := models.Agent{
		Name:        body.Name,
		OwnerUserID: claims.UserID,
		SkillID:     body.SkillID,
	}
	if err := h.db.Create(&agent).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	apiKey, err := h.apiKeys.Create(agent.ID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusCreated, gin.H{"agent": agent, "api_key": apiKey})
}

func (h *Handlers) rotateAgentKey(c *gin.Context) {
	claims := GetClaims(c)
	id, err := uuid.Parse(c.Param("id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid id"})
		return
	}
	var agent models.Agent
	if err := h.db.First(&agent, "id = ?", id).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "not found"})
		return
	}
	if claims.Role != "admin" && agent.OwnerUserID != claims.UserID {
		c.JSON(http.StatusForbidden, gin.H{"error": "forbidden"})
		return
	}
	_ = h.db.Where("agent_id = ?", id).Delete(&models.APIKey{}).Error
	apiKey, err := h.apiKeys.Create(id)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, gin.H{"api_key": apiKey})
}

func queryInt(c *gin.Context, key string, fallback int) int {
	v := c.Query(key)
	if v == "" {
		return fallback
	}
	n, err := strconv.Atoi(v)
	if err != nil {
		return fallback
	}
	return n
}

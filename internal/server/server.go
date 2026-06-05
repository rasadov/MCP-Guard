package server

import (
	"embed"
	"io/fs"
	"log/slog"
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/rasadov/mcp-guard/internal/api"
	"github.com/rasadov/mcp-guard/internal/audit"
	"github.com/rasadov/mcp-guard/internal/auth"
	"github.com/rasadov/mcp-guard/internal/config"
	mcpgw "github.com/rasadov/mcp-guard/internal/mcp"
	"github.com/rasadov/mcp-guard/internal/policy"
	"github.com/rasadov/mcp-guard/internal/shadow"
	"gorm.io/gorm"
)

type Server struct {
	cfg        config.Config
	db         *gorm.DB
	r          *gin.Engine
	downstream *mcpgw.Downstream
}

func New(cfg config.Config, db *gorm.DB, webFS embed.FS, downstream *mcpgw.Downstream) *Server {
	gin.SetMode(gin.ReleaseMode)
	r := gin.New()
	r.Use(gin.Recovery())
	r.Use(requestLogger())

	apiKeys := auth.NewAPIKeyService(db)
	jwtSvc := auth.NewJWTService(cfg.JWTSecret, cfg.JWTExpiry)
	googleAuth := auth.NewGoogleAuth(cfg.Google)
	policyEngine := policy.NewEngine()
	auditSvc := audit.NewService(db)
	shadowSvc := shadow.NewService(db)

	toolList := func() []string {
		if downstream == nil {
			return nil
		}
		var names []string
		for name := range downstream.Tools() {
			names = append(names, name)
		}
		return names
	}

	gateway := mcpgw.NewGateway(db, downstream, apiKeys, policyEngine, auditSvc)
	handlers := api.NewHandlers(db, cfg, jwtSvc, googleAuth, apiKeys, auditSvc, shadowSvc, policyEngine, toolList)

	s := &Server{cfg: cfg, db: db, r: r, downstream: downstream}
	s.registerRoutes(webFS, gateway, handlers)
	return s
}

func (s *Server) Run() error {
	slog.Info("starting gateway", "addr", s.cfg.Addr)
	return s.r.Run(s.cfg.Addr)
}

func (s *Server) Engine() *gin.Engine {
	return s.r
}

func (s *Server) registerRoutes(webFS embed.FS, gateway *mcpgw.Gateway, handlers *api.Handlers) {
	s.r.GET("/healthz", func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{"status": "ok"})
	})

	s.r.Any("/mcp", gin.WrapH(gateway.Handler()))
	s.r.Any("/mcp/*path", gin.WrapH(gateway.Handler()))

	authGroup := s.r.Group("/auth")
	handlers.RegisterAuth(authGroup)

	apiGroup := s.r.Group("/api/v1")
	handlers.Register(apiGroup)

	if sub, err := fs.Sub(webFS, "dist"); err == nil {
		static := http.FileServer(http.FS(sub))
		s.r.NoRoute(func(c *gin.Context) {
			if c.Request.Method != http.MethodGet {
				c.JSON(http.StatusNotFound, gin.H{"error": "not found"})
				return
			}
			path := c.Request.URL.Path
			if path == "/" || path == "" {
				c.Request.URL.Path = "/index.html"
			}
			static.ServeHTTP(c.Writer, c.Request)
		})
	} else {
		s.r.GET("/", func(c *gin.Context) {
			c.JSON(http.StatusOK, gin.H{
				"service": "mcp-guard",
				"message": "dashboard assets not embedded yet",
			})
		})
	}
}

func requestLogger() gin.HandlerFunc {
	return func(c *gin.Context) {
		c.Next()
		slog.Info("request",
			"method", c.Request.Method,
			"path", c.Request.URL.Path,
			"status", c.Writer.Status(),
		)
	}
}

package server

import (
	"embed"
	"io/fs"
	"log/slog"
	"net/http"
	"path/filepath"
	"strings"

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
	r.RedirectTrailingSlash = false
	r.RedirectFixedPath = false
	r.Use(gin.Recovery())
	r.Use(requestLogger())

	apiKeys := auth.NewAPIKeyService(db)
	jwtSvc := auth.NewJWTService(cfg.JWTSecret, cfg.JWTExpiry)
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
	handlers := api.NewHandlers(db, cfg, jwtSvc, apiKeys, auditSvc, shadowSvc, policyEngine, toolList)

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

	if dist, err := fs.Sub(webFS, "dist"); err == nil {
		serveDist := func(c *gin.Context) {
			if c.Request.Method != http.MethodGet && c.Request.Method != http.MethodHead {
				c.JSON(http.StatusNotFound, gin.H{"error": "not found"})
				return
			}
			reqPath := strings.TrimPrefix(c.Request.URL.Path, "/")
			if reqPath == "" {
				reqPath = "index.html"
			}
			if strings.Contains(reqPath, "..") {
				c.JSON(http.StatusNotFound, gin.H{"error": "not found"})
				return
			}
			data, err := fs.ReadFile(dist, reqPath)
			if err != nil {
				data, err = fs.ReadFile(dist, "index.html")
				if err != nil {
					c.JSON(http.StatusNotFound, gin.H{"error": "not found"})
					return
				}
				reqPath = "index.html"
			}
			c.Data(http.StatusOK, contentType(reqPath), data)
		}

		s.r.GET("/", serveDist)
		s.r.NoRoute(func(c *gin.Context) {
			p := c.Request.URL.Path
			if strings.HasPrefix(p, "/api/") || strings.HasPrefix(p, "/auth/") || strings.HasPrefix(p, "/mcp") || p == "/healthz" {
				c.JSON(http.StatusNotFound, gin.H{"error": "not found"})
				return
			}
			serveDist(c)
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

func contentType(name string) string {
	switch filepath.Ext(name) {
	case ".html":
		return "text/html; charset=utf-8"
	case ".js":
		return "application/javascript"
	case ".css":
		return "text/css; charset=utf-8"
	case ".svg":
		return "image/svg+xml"
	case ".png":
		return "image/png"
	default:
		return "application/octet-stream"
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

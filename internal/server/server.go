package server

import (
	"embed"
	"io/fs"
	"log/slog"
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/rasadov/mcp-guard/internal/config"
	"gorm.io/gorm"
)

type Server struct {
	cfg config.Config
	db  *gorm.DB
	r   *gin.Engine
}

func New(cfg config.Config, db *gorm.DB, webFS embed.FS) *Server {
	gin.SetMode(gin.ReleaseMode)
	r := gin.New()
	r.Use(gin.Recovery())
	r.Use(requestLogger())

	s := &Server{cfg: cfg, db: db, r: r}
	s.registerRoutes(webFS)
	return s
}

func (s *Server) Run() error {
	slog.Info("starting gateway", "addr", s.cfg.Addr)
	return s.r.Run(s.cfg.Addr)
}

func (s *Server) Engine() *gin.Engine {
	return s.r
}

func (s *Server) registerRoutes(webFS embed.FS) {
	s.r.GET("/healthz", func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{"status": "ok"})
	})

	api := s.r.Group("/api/v1")
	api.GET("/status", func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{"service": "mcp-guard", "version": "0.1.0"})
	})

	if sub, err := fs.Sub(webFS, "dist"); err == nil {
		s.r.NoRoute(func(c *gin.Context) {
			if c.Request.Method != http.MethodGet {
				c.JSON(http.StatusNotFound, gin.H{"error": "not found"})
				return
			}
			http.FileServer(http.FS(sub)).ServeHTTP(c.Writer, c.Request)
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

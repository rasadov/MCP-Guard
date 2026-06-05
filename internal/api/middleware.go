package api

import (
	"net/http"
	"strings"

	"github.com/gin-gonic/gin"
	"github.com/rasadov/mcp-guard/internal/auth"
)

const userContextKey = "user_claims"

func JWTMiddleware(jwtSvc *auth.JWTService) gin.HandlerFunc {
	return func(c *gin.Context) {
		token := ""
		if cookie, err := c.Cookie("mcp_guard_token"); err == nil {
			token = cookie
		}
		if token == "" {
			authHeader := c.GetHeader("Authorization")
			if strings.HasPrefix(authHeader, "Bearer ") {
				token = strings.TrimPrefix(authHeader, "Bearer ")
			}
		}
		if token == "" {
			c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{"error": "unauthorized"})
			return
		}
		claims, err := jwtSvc.Parse(token)
		if err != nil {
			c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{"error": "invalid token"})
			return
		}
		c.Set(userContextKey, claims)
		c.Next()
	}
}

func RequireAdmin() gin.HandlerFunc {
	return func(c *gin.Context) {
		claims, ok := c.Get(userContextKey)
		if !ok {
			c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{"error": "unauthorized"})
			return
		}
		userClaims := claims.(*auth.Claims)
		if userClaims.Role != "admin" {
			c.AbortWithStatusJSON(http.StatusForbidden, gin.H{"error": "admin required"})
			return
		}
		c.Next()
	}
}

func GetClaims(c *gin.Context) *auth.Claims {
	v, _ := c.Get(userContextKey)
	if v == nil {
		return nil
	}
	return v.(*auth.Claims)
}

package middleware

import (
	"github.com/MicahParks/keyfunc/v2"
	"github.com/gin-gonic/gin"
	"github.com/golang-jwt/jwt/v5"
	"log"
	"net/http"
	"strings"
	"time"
)

const (
	ContextUserIDKey = "userID"
	ContextScopesKey = "scopes"
)

type CustomClaims struct {
	jwt.RegisteredClaims
	Scopes string `json:"scope,omitempty"`
}

type AuthHandler interface {
	AuthMiddleware() gin.HandlerFunc
}

type authHandler struct {
	jwks *keyfunc.JWKS
}

func NewAuthHandler(jwksURL string) (AuthHandler, error) {
	options := keyfunc.Options{
		RefreshErrorHandler: func(err error) {
			log.Printf("There was an error with the jwt.Keyfunc\nError: %s", err.Error())
		},
		RefreshInterval:   time.Hour,
		RefreshRateLimit:  time.Minute * 5,
		RefreshTimeout:    time.Second * 10,
		RefreshUnknownKID: true,
	}

	jwks, err := keyfunc.Get(jwksURL, options)
	if err != nil {
		log.Fatalf("Failed to create JWKS from resource at the given URL.\nError: %s", err.Error())
	}

	return &authHandler{jwks: jwks}, nil
}

func (h *authHandler) AuthMiddleware() gin.HandlerFunc {
	return func(c *gin.Context) {
		if c.Request.URL.Path == "/health" {
			c.Next()
			return
		}
		tokenString := c.GetHeader("Authorization")
		if tokenString == "" {
			c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{"error": "authorization header is required"})
			return
		}

		tokenString = strings.TrimPrefix(tokenString, "Bearer ")

		var claims CustomClaims
		token, err := jwt.ParseWithClaims(tokenString, &claims, h.jwks.Keyfunc)
		if err != nil || !token.Valid {
			c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{"error": "invalid token"})
			return
		}

		if token.Valid {
			scopes := strings.Split(claims.Scopes, " ")
			c.Set(ContextUserIDKey, claims.Subject)
			c.Set(ContextScopesKey, scopes)
		} else {
			c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{"error": "invalid token claims"})
			return
		}

		c.Next()
	}
}

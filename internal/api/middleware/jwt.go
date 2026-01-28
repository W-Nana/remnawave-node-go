package middleware

import (
	"crypto/rsa"
	"crypto/x509"
	"encoding/pem"
	"fmt"
	"net/http"
	"strings"

	"github.com/gin-gonic/gin"
	"github.com/golang-jwt/jwt/v5"

	"github.com/remnawave/node-go/internal/logger"
)

// JWTMiddleware creates a middleware that validates JWT tokens using RS256.
// On auth failure, the socket is destroyed (no HTTP response sent).
// This matches the original NestJS behavior: response.socket?.destroy()
func JWTMiddleware(publicKeyPEM string, log *logger.Logger) gin.HandlerFunc {
	// Parse the RSA public key once at initialization
	publicKey, err := parseRSAPublicKey(publicKeyPEM)
	if err != nil {
		// If key parsing fails at startup, return middleware that always fails
		return func(c *gin.Context) {
			if log != nil {
				log.Error(fmt.Sprintf("JWT middleware disabled: invalid public key: %v", err))
			}
			destroySocket(c)
		}
	}

	return func(c *gin.Context) {
		// Extract token from Authorization header
		authHeader := c.GetHeader("Authorization")
		if authHeader == "" {
			logAuthFailure(log, c, "missing Authorization header")
			destroySocket(c)
			return
		}

		// Expect "Bearer <token>" format
		parts := strings.SplitN(authHeader, " ", 2)
		if len(parts) != 2 || !strings.EqualFold(parts[0], "Bearer") {
			logAuthFailure(log, c, "invalid Authorization header format")
			destroySocket(c)
			return
		}

		tokenString := parts[1]

		// Parse and validate token
		token, err := jwt.Parse(tokenString, func(token *jwt.Token) (interface{}, error) {
			// Verify signing method is RS256
			if _, ok := token.Method.(*jwt.SigningMethodRSA); !ok {
				return nil, fmt.Errorf("unexpected signing method: %v", token.Header["alg"])
			}
			return publicKey, nil
		}, jwt.WithValidMethods([]string{"RS256"}))

		if err != nil {
			logAuthFailure(log, c, fmt.Sprintf("token validation failed: %v", err))
			destroySocket(c)
			return
		}

		if !token.Valid {
			logAuthFailure(log, c, "invalid token")
			destroySocket(c)
			return
		}

		// Token is valid - store claims in context for later use
		if claims, ok := token.Claims.(jwt.MapClaims); ok {
			c.Set("jwt_claims", claims)
		}

		c.Next()
	}
}

// parseRSAPublicKey parses a PEM-encoded RSA public key.
func parseRSAPublicKey(publicKeyPEM string) (*rsa.PublicKey, error) {
	block, _ := pem.Decode([]byte(publicKeyPEM))
	if block == nil {
		return nil, fmt.Errorf("failed to parse PEM block")
	}

	// Try parsing as PKIX (standard format)
	pub, err := x509.ParsePKIXPublicKey(block.Bytes)
	if err != nil {
		// Try parsing as PKCS1 format
		rsaPub, err := x509.ParsePKCS1PublicKey(block.Bytes)
		if err != nil {
			return nil, fmt.Errorf("failed to parse public key: %w", err)
		}
		return rsaPub, nil
	}

	rsaPub, ok := pub.(*rsa.PublicKey)
	if !ok {
		return nil, fmt.Errorf("not an RSA public key")
	}

	return rsaPub, nil
}

// logAuthFailure logs authentication failure with request details.
func logAuthFailure(log *logger.Logger, c *gin.Context, reason string) {
	if log != nil {
		log.WithField("url", c.Request.URL.String()).
			WithField("ip", c.ClientIP()).
			WithField("reason", reason).
			Error("Incorrect SECRET_KEY or JWT! Request dropped.")
	}
}

// destroySocket forcefully closes the underlying TCP connection.
// This matches NestJS behavior: response.socket?.destroy()
func destroySocket(c *gin.Context) {
	defer func() {
		recover()
		c.Abort()
	}()

	hijacker, ok := c.Writer.(http.Hijacker)
	if !ok {
		return
	}
	conn, _, err := hijacker.Hijack()
	if err != nil {
		return
	}
	conn.Close()
}

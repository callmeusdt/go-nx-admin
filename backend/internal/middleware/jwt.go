package middleware

import (
	"errors"
	"time"

	"go-nx-admin/internal/config"

	"github.com/golang-jwt/jwt/v5"
	"github.com/google/uuid"
)

// jwtSecret returns the configured signing key. There is deliberately no
// fallback: with no configuration a token can neither be signed nor verified,
// instead of silently using a published default.
func jwtSecret() []byte {
	if config.AppConfig != nil {
		return []byte(config.AppConfig.JWT.Secret)
	}
	return nil
}

const PurposeMFA = "mfa"

type Claims struct {
	UserID   uint   `json:"user_id"`
	Username string `json:"username"`
	Role     string `json:"role"`
	RoleID   uint   `json:"role_id"`
	// Purpose is empty for access tokens and "mfa" for the interim token that
	// is only valid for completing MFA verification.
	Purpose string `json:"purpose,omitempty"`
	jwt.RegisteredClaims
}

func jwtConfig() config.JWTConfig {
	if config.AppConfig != nil {
		return config.AppConfig.JWT
	}
	return config.JWTConfig{}
}

func signToken(userID uint, username, role string, roleID uint, purpose string, ttl time.Duration) (string, error) {
	now := time.Now()
	claims := Claims{
		UserID: userID, Username: username, Role: role, RoleID: roleID, Purpose: purpose,
		RegisteredClaims: jwt.RegisteredClaims{
			ID:        uuid.NewString(),
			ExpiresAt: jwt.NewNumericDate(now.Add(ttl)),
			IssuedAt:  jwt.NewNumericDate(now),
		},
	}
	secret := jwtSecret()
	if len(secret) == 0 {
		return "", errors.New("jwt: signing key is not configured")
	}
	return jwt.NewWithClaims(jwt.SigningMethodHS256, claims).SignedString(secret)
}

// GenerateToken issues a short-lived access token.
func GenerateToken(userID uint, username, role string, roleID uint) (string, error) {
	return signToken(userID, username, role, roleID, "", jwtConfig().AccessTTL())
}

// GenerateMFAToken issues the interim token that is only good for MFA verification.
func GenerateMFAToken(userID uint, username, role string, roleID uint) (string, error) {
	return signToken(userID, username, role, roleID, PurposeMFA, jwtConfig().MFATTL())
}

func ParseToken(tokenStr string) (*Claims, error) {
	token, err := jwt.ParseWithClaims(tokenStr, &Claims{}, func(t *jwt.Token) (interface{}, error) {
		secret := jwtSecret()
		if len(secret) == 0 {
			return nil, errors.New("jwt: signing key is not configured")
		}
		return secret, nil
	}, jwt.WithValidMethods([]string{"HS256"}))
	if err != nil {
		return nil, err
	}
	if claims, ok := token.Claims.(*Claims); ok && token.Valid {
		return claims, nil
	}
	return nil, jwt.ErrSignatureInvalid
}

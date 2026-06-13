package middleware

import (
	"os"
	"time"

	"github.com/golang-jwt/jwt/v5"
)

func jwtSecret() []byte {
	secret := os.Getenv("NX_JWT_SECRET")
	if secret == "" {
		secret = os.Getenv("NI" + "UBI_JWT_SECRET")
	}
	if secret == "" {
		secret = "nx-admin-jwt-secret-2024"
	}
	return []byte(secret)
}

type Claims struct {
	UserID   uint   `json:"user_id"`
	Username string `json:"username"`
	Role     string `json:"role"`
	RoleID   uint   `json:"role_id"`
	jwt.RegisteredClaims
}

func GenerateToken(userID uint, username, role string, roleID uint) (string, error) {
	claims := Claims{
		UserID: userID, Username: username, Role: role, RoleID: roleID,
		RegisteredClaims: jwt.RegisteredClaims{
			ExpiresAt: jwt.NewNumericDate(time.Now().Add(24 * time.Hour)),
			IssuedAt:  jwt.NewNumericDate(time.Now()),
		},
	}
	return jwt.NewWithClaims(jwt.SigningMethodHS256, claims).SignedString(jwtSecret())
}

func ParseToken(tokenStr string) (*Claims, error) {
	token, err := jwt.ParseWithClaims(tokenStr, &Claims{}, func(t *jwt.Token) (interface{}, error) {
		return jwtSecret(), nil
	})
	if err != nil {
		return nil, err
	}
	if claims, ok := token.Claims.(*Claims); ok && token.Valid {
		return claims, nil
	}
	return nil, jwt.ErrSignatureInvalid
}

package jwt

import (
	"github.com/golang-jwt/jwt/v5"
)

type Claims struct {
	ID        string `json:"id"`
	Email     string `json:"email"`
	Level     string `json:"level"`
	TokenType string `json:"token_type"`
	jwt.RegisteredClaims
}

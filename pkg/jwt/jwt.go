package jwt

import (
	"errors"
	"fmt"
	"sync"
	"time"

	"github.com/golang-jwt/jwt/v5"
)

var (
	instance *JWT
	once     sync.Once

	ErrJWTNotInitialized = errors.New("jwt: instance not initialized")
	ErrInvalidToken      = errors.New("jwt: invalid token")
)

type JWT struct {
	appName            string
	secretKey          string
	accessTokenExpiry  time.Duration
	refreshTokenExpiry time.Duration
}

func Initialize(appName string, secretKey string, accessExpiry, refreshExpiry time.Duration) {
	once.Do(func() {
		instance = &JWT{
			appName:            appName,
			secretKey:          secretKey,
			accessTokenExpiry:  accessExpiry,
			refreshTokenExpiry: refreshExpiry,
		}
	})
}

func GetInstance() *JWT {
	if instance == nil {
		_ = ErrJWTNotInitialized
	}

	return instance
}

func GenerateAccessToken(userID, email, level string) (string, error) {
	return GetInstance().generateToken(userID, email, level, GetInstance().accessTokenExpiry, "access_token")
}

func GenerateRefreshToken(userID, email, level string) (string, error) {
	return GetInstance().generateToken(userID, email, level, GetInstance().refreshTokenExpiry, "refresh_token")
}

func ValidateToken(tokenString string) (*Claims, error) {
	token, err := jwt.ParseWithClaims(tokenString, &Claims{}, func(_ *jwt.Token) (interface{}, error) {
		return []byte(GetInstance().secretKey), nil
	})
	if err != nil {
		return nil, fmt.Errorf("failed to parse token: %w", err)
	}

	if claims, ok := token.Claims.(*Claims); ok && token.Valid {
		return claims, nil
	}

	return nil, ErrInvalidToken
}

func (j *JWT) generateToken(userID, email, level string, expiry time.Duration, tokenType string) (string, error) {
	claims := &Claims{
		ID:        userID,
		Email:     email,
		Level:     level,
		TokenType: tokenType,
		RegisteredClaims: jwt.RegisteredClaims{
			ExpiresAt: jwt.NewNumericDate(time.Now().Add(expiry)),
			IssuedAt:  jwt.NewNumericDate(time.Now()),
			NotBefore: jwt.NewNumericDate(time.Now()),
			Issuer:    j.appName,
			Subject:   userID,
		},
	}

	token := jwt.NewWithClaims(jwt.SigningMethodHS512, claims)

	signedString, err := token.SignedString([]byte(j.secretKey))
	if err != nil {
		return "", fmt.Errorf("jwt: failed to sign token: %w", err)
	}

	return signedString, nil
}

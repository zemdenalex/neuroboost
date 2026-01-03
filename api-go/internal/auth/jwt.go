package auth

import (
	"errors"
	"fmt"
	"os"
	"sync"
	"time"

	"github.com/golang-jwt/jwt/v5"
)

const tokenTTL = 30 * 24 * time.Hour
const refreshWindow = 7 * 24 * time.Hour

var (
	secretOnce sync.Once
	jwtSecret  []byte
	secretErr  error
)

func loadSecret() error {
	secretOnce.Do(func() {
		s := os.Getenv("JWT_SECRET")
		if s == "" {
			secretErr = errors.New("JWT_SECRET is not set")
			return
		}
		jwtSecret = []byte(s)
	})
	return secretErr
}

func GenerateToken(userID int64, telegramID int64, username string) (string, error) {
	if err := loadSecret(); err != nil {
		return "", err
	}
	now := time.Now()
	exp := now.Add(tokenTTL)

	claims := &Claims{
		UserID:     userID,
		TelegramID: telegramID,
		Username:   username,
		RegisteredClaims: jwt.RegisteredClaims{
			IssuedAt:  jwt.NewNumericDate(now),
			ExpiresAt: jwt.NewNumericDate(exp),
		},
	}

	t := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	signed, err := t.SignedString(jwtSecret)
	if err != nil {
		return "", fmt.Errorf("sign jwt: %w", err)
	}
	return signed, nil
}

func ValidateToken(tokenString string) (*Claims, error) {
	if err := loadSecret(); err != nil {
		return nil, err
	}

	parser := jwt.NewParser(jwt.WithValidMethods([]string{jwt.SigningMethodHS256.Alg()}))
	token, err := parser.ParseWithClaims(tokenString, &Claims{}, func(t *jwt.Token) (interface{}, error) {
		return jwtSecret, nil
	})
	if err != nil {
		return nil, fmt.Errorf("parse jwt: %w", err)
	}
	claims, ok := token.Claims.(*Claims)
	if !ok || !token.Valid {
		return nil, errors.New("invalid token claims")
	}
	if claims.ExpiresAt == nil || time.Now().After(claims.ExpiresAt.Time) {
		return nil, errors.New("token expired")
	}
	return claims, nil
}

func RefreshToken(tokenString string) (string, error) {
	claims, err := ValidateToken(tokenString)
	if err != nil {
		return "", err
	}
	if time.Until(claims.ExpiresAt.Time) < refreshWindow {
		return GenerateToken(claims.UserID, claims.TelegramID, claims.Username)
	}
	return tokenString, nil
}

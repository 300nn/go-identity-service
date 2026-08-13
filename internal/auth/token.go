package auth

import (
	"fmt"
	"time"

	"github.com/golang-jwt/jwt/v5"
)

type TokenManager struct {
	secret []byte
	ttl    time.Duration
	issuer string
}

func NewTokenManager(secret string, ttl time.Duration, issuer string) *TokenManager {
	return &TokenManager{
		secret: []byte(secret),
		ttl:    ttl,
		issuer: issuer,
	}
}

type Claims struct {
	UserId int64  `json:"userId"`
	Email  string `json:"email"`
	Role   string `json:"role"`

	jwt.RegisteredClaims
}

func (m *TokenManager) Generate(userId int64, email string, role string) (string, error) {
	now := time.Now().UTC()

	claims := Claims{
		UserId: userId,
		Email:  email,
		Role:   role,
		RegisteredClaims: jwt.RegisteredClaims{
			Issuer:    m.issuer,
			Subject:   fmt.Sprintf("%d", userId),
			IssuedAt:  jwt.NewNumericDate(now),
			ExpiresAt: jwt.NewNumericDate(now.Add(m.ttl)),
		},
	}

	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)

	signed, err := token.SignedString(m.secret)
	if err != nil {
		return "", fmt.Errorf("sign jwt: %w", err)
	}

	return signed, nil
}

func (m *TokenManager) Parse(tokenString string) (Claims, error) {
	token, err := jwt.ParseWithClaims(
		tokenString,
		&Claims{},
		func(token *jwt.Token) (any, error) {
			if token.Method != jwt.SigningMethodHS256 {
				return nil, fmt.Errorf("unexpected signing method: %s", token.Method.Alg())
			}

			return m.secret, nil
		},
		jwt.WithIssuer(m.issuer),
		jwt.WithExpirationRequired(),
	)

	if err != nil {
		return Claims{}, fmt.Errorf("parse jwt: %w", err)
	}

	claims, ok := token.Claims.(*Claims)

	if !ok || !token.Valid {
		return Claims{}, ErrUnauthorized
	}

	return *claims, nil
}

package auth

import (
	"errors"
	"fmt"
	"time"

	"github.com/golang-jwt/jwt/v5"
	"github.com/google/uuid"
	"github.com/m161awm2/velocity-bulletinBE/internal/model"
)

type Manager struct {
	secret []byte
	ttl    time.Duration
}

type Claims struct {
	Role model.Role `json:"role"`
	jwt.RegisteredClaims
}

func New(secret string, ttl time.Duration) *Manager {
	return &Manager{secret: []byte(secret), ttl: ttl}
}

func (m *Manager) Issue(user *model.User) (string, time.Time, error) {
	now := time.Now().UTC()
	expiresAt := now.Add(m.ttl)
	claims := Claims{
		Role: user.Role,
		RegisteredClaims: jwt.RegisteredClaims{
			Subject: user.ID.String(), IssuedAt: jwt.NewNumericDate(now),
			ExpiresAt: jwt.NewNumericDate(expiresAt), Issuer: "velocity-bulletin",
		},
	}
	token, err := jwt.NewWithClaims(jwt.SigningMethodHS256, claims).SignedString(m.secret)
	return token, expiresAt, err
}

func (m *Manager) Parse(raw string) (uuid.UUID, error) {
	token, err := jwt.ParseWithClaims(raw, &Claims{}, func(token *jwt.Token) (any, error) {
		if token.Method != jwt.SigningMethodHS256 {
			return nil, fmt.Errorf("unexpected signing method: %s", token.Method.Alg())
		}
		return m.secret, nil
	}, jwt.WithIssuer("velocity-bulletin"), jwt.WithExpirationRequired())
	if err != nil || !token.Valid {
		return uuid.Nil, errors.New("invalid token")
	}
	claims, ok := token.Claims.(*Claims)
	if !ok {
		return uuid.Nil, errors.New("invalid claims")
	}
	return uuid.Parse(claims.Subject)
}

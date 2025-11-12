package token

import (
	"errors"
	"time"

	"github.com/changhyeonkim/pray-together/go-api-server/internal/config"
	"github.com/golang-jwt/jwt/v5"
)

var (
	ErrInvalidToken  = errors.New("token: invalid token")
	ErrExpiredToken  = errors.New("token: expired token")
	ErrInvalidClaims = errors.New("token: invalid claims")
)

const (
	ACCESS  = "access"
	REFRESH = "refresh"
)

type Claims struct {
	MemberID  string `json:"member_id"`
	Email     string `json:"email"`
	TokenType string `json:"token_type"`
	ExpiresAt int64  `json:"exp"`
	IssuedAt  int64  `json:"iat"`
	jwt.RegisteredClaims
}

type Manager interface {
	GenerateAccessToken(memberID string, email string) (string, error)
	GenerateRefreshToken(memerID string, email string) (string, error)
	ValidateToken(tokenString string) (*Claims, error)
}

type JWTManager struct {
	secret        []byte
	issuer        string
	accessExpiry  time.Duration
	refreshExpiry time.Duration
}

func NewJWTManager(cfg *config.Config) *JWTManager {
	return &JWTManager{
		secret:        []byte(cfg.JWT.Secret),
		issuer:        cfg.App.Name,
		accessExpiry:  cfg.JWT.Expiry,
		refreshExpiry: cfg.JWT.RefreshExpiry,
	}
}

func (m *JWTManager) GenerateAccessToken(memberID, email string) (string, error) {
	now := time.Now()
	expiresAt := now.Add(m.accessExpiry)

	claims := Claims{
		MemberID:  memberID,
		Email:     email,
		ExpiresAt: expiresAt.Unix(),
		TokenType: ACCESS,
		IssuedAt:  now.Unix(),
		RegisteredClaims: jwt.RegisteredClaims{
			ExpiresAt: jwt.NewNumericDate(expiresAt),
			IssuedAt:  jwt.NewNumericDate(now),
			Issuer:    m.issuer,
		},
	}

	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	return token.SignedString(m.secret)
}

func (m *JWTManager) GenerateRefreshToken(memberID string, email string) (string, error) {
	now := time.Now()
	expiresAt := now.Add(m.refreshExpiry)

	claims := Claims{
		MemberID:  memberID,
		Email:     email,
		TokenType: REFRESH,
		ExpiresAt: expiresAt.Unix(),
		IssuedAt:  now.Unix(),
		RegisteredClaims: jwt.RegisteredClaims{
			Subject:   memberID,
			ExpiresAt: jwt.NewNumericDate(expiresAt),
			IssuedAt:  jwt.NewNumericDate(now),
			Issuer:    m.issuer,
		},
	}

	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	return token.SignedString(m.secret)
}

func (m *JWTManager) ValidateToken(tokenString string) (*Claims, error) {
	parser := jwt.NewParser(jwt.WithValidMethods([]string{jwt.SigningMethodHS256.Name}))

	token, err := parser.ParseWithClaims(tokenString, &Claims{}, func(token *jwt.Token) (interface{}, error) {
		return m.secret, nil
	})
	if err != nil {
		if errors.Is(err, jwt.ErrTokenExpired) {
			return nil, ErrExpiredToken
		}
		return nil, ErrInvalidToken
	}

	claims, ok := token.Claims.(*Claims)
	if !ok {
		return nil, ErrInvalidClaims
	}

	if !token.Valid {
		return nil, ErrInvalidToken
	}

	return claims, nil
}

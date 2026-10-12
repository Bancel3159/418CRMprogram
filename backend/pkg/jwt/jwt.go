package jwt

import (
	"errors"
	"fmt"
	"time"

	"github.com/golang-jwt/jwt/v5"
)

// 错误定义
var (
	ErrInvalidToken     = errors.New("invalid token")
	ErrTokenExpired     = errors.New("token expired")
	ErrTokenNotYetValid = errors.New("token not yet valid")
)

// Claims 自定义 JWT 载荷
type Claims struct {
	UserID   uint   `json:"uid"`
	Username string `json:"username"`
	Role     string `json:"role"`
	DeptID   *uint  `json:"did,omitempty"`
	Type     string `json:"type"` // access / refresh
	jwt.RegisteredClaims
}

// Manager JWT 管理器
type Manager struct {
	secret          []byte
	accessTokenTTL  time.Duration
	refreshTokenTTL time.Duration
}

// NewManager 创建 JWT 管理器
func NewManager(secret string, accessTTL, refreshTTL int) *Manager {
	return &Manager{
		secret:          []byte(secret),
		accessTokenTTL:  time.Duration(accessTTL) * time.Second,
		refreshTokenTTL: time.Duration(refreshTTL) * time.Second,
	}
}

// GenerateAccessToken 生成 access token
func (m *Manager) GenerateAccessToken(userID uint, username, role string, deptID *uint) (string, time.Time, error) {
	return m.generate(userID, username, role, deptID, "access", m.accessTokenTTL)
}

// GenerateRefreshToken 生成 refresh token
func (m *Manager) GenerateRefreshToken(userID uint, username, role string, deptID *uint) (string, time.Time, error) {
	return m.generate(userID, username, role, deptID, "refresh", m.refreshTokenTTL)
}

func (m *Manager) generate(userID uint, username, role string, deptID *uint, tokenType string, ttl time.Duration) (string, time.Time, error) {
	expiresAt := time.Now().Add(ttl)
	claims := Claims{
		UserID:   userID,
		Username: username,
		Role:     role,
		DeptID:   deptID,
		Type:     tokenType,
		RegisteredClaims: jwt.RegisteredClaims{
			ExpiresAt: jwt.NewNumericDate(expiresAt),
			IssuedAt:  jwt.NewNumericDate(time.Now()),
			NotBefore: jwt.NewNumericDate(time.Now()),
			Issuer:    "crm-backend",
			Subject:   fmt.Sprintf("%d", userID),
		},
	}

	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	signed, err := token.SignedString(m.secret)
	if err != nil {
		return "", time.Time{}, err
	}
	return signed, expiresAt, nil
}

// Parse 解析并验证 token
func (m *Manager) Parse(tokenStr string) (*Claims, error) {
	claims := &Claims{}
	token, err := jwt.ParseWithClaims(tokenStr, claims, func(t *jwt.Token) (interface{}, error) {
		if _, ok := t.Method.(*jwt.SigningMethodHMAC); !ok {
			return nil, fmt.Errorf("unexpected signing method: %v", t.Header["alg"])
		}
		return m.secret, nil
	})

	if err != nil {
		if errors.Is(err, jwt.ErrTokenExpired) {
			return nil, ErrTokenExpired
		}
		if errors.Is(err, jwt.ErrTokenNotValidYet) {
			return nil, ErrTokenNotYetValid
		}
		return nil, ErrInvalidToken
	}

	if !token.Valid {
		return nil, ErrInvalidToken
	}

	return claims, nil
}

// ParseAccessToken 解析 access token
func (m *Manager) ParseAccessToken(tokenStr string) (*Claims, error) {
	claims, err := m.Parse(tokenStr)
	if err != nil {
		return nil, err
	}
	if claims.Type != "access" {
		return nil, ErrInvalidToken
	}
	return claims, nil
}

// ParseRefreshToken 解析 refresh token
func (m *Manager) ParseRefreshToken(tokenStr string) (*Claims, error) {
	claims, err := m.Parse(tokenStr)
	if err != nil {
		return nil, err
	}
	if claims.Type != "refresh" {
		return nil, ErrInvalidToken
	}
	return claims, nil
}
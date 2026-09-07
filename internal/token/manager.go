package token

import (
	"crypto/hmac"
	"crypto/rand"
	"crypto/sha256"
	"encoding/base64"
	"encoding/json"
	"strings"
	"time"

	"friends-records/internal/apperror"
)

var ErrInvalid = apperror.New("登录令牌无效或已经过期")

type Claims struct {
	Kind      string `json:"kind"`
	SubjectID uint64 `json:"sub"`
	ExpiresAt int64  `json:"exp"`
}

type Manager struct {
	secret []byte
}

func New(secret string) *Manager {
	return &Manager{secret: []byte(secret)}
}

func NewRandom() (*Manager, error) {
	secret := make([]byte, 32)
	if _, err := rand.Read(secret); err != nil {
		return nil, apperror.Wrap(err, "生成开发环境临时 Token 密钥失败")
	}
	return &Manager{secret: secret}, nil
}

func (m *Manager) Issue(kind string, subjectID uint64, ttl time.Duration) (string, error) {
	claims := Claims{Kind: kind, SubjectID: subjectID, ExpiresAt: time.Now().Add(ttl).Unix()}
	payload, err := json.Marshal(claims)
	if err != nil {
		return "", apperror.Wrap(err, "生成登录令牌失败")
	}
	encoded := base64.RawURLEncoding.EncodeToString(payload)
	return encoded + "." + m.signature(encoded), nil
}

func (m *Manager) Parse(value, expectedKind string) (Claims, error) {
	parts := strings.Split(value, ".")
	if len(parts) != 2 || !hmac.Equal([]byte(parts[1]), []byte(m.signature(parts[0]))) {
		return Claims{}, ErrInvalid
	}
	payload, err := base64.RawURLEncoding.DecodeString(parts[0])
	if err != nil {
		return Claims{}, ErrInvalid
	}
	var claims Claims
	if err := json.Unmarshal(payload, &claims); err != nil {
		return Claims{}, ErrInvalid
	}
	if claims.Kind != expectedKind || claims.SubjectID == 0 || time.Now().Unix() >= claims.ExpiresAt {
		return Claims{}, ErrInvalid
	}
	return claims, nil
}

func (m *Manager) signature(payload string) string {
	mac := hmac.New(sha256.New, m.secret)
	mac.Write([]byte(payload))
	return base64.RawURLEncoding.EncodeToString(mac.Sum(nil))
}

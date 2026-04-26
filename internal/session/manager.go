package session

import (
	"crypto/hmac"
	"crypto/sha256"
	"encoding/base64"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"strings"
	"time"
)

const (
	// DefaultCookieName 默认用户会话 Cookie 名称
	DefaultCookieName = "donation_session"
	defaultTTL        = 7 * 24 * time.Hour
)

var (
	errInvalidToken = errors.New("无效的登录状态")
	errExpiredToken = errors.New("登录状态已过期")
)

// Claims 当前站点保存的登录态信息
type Claims struct {
	LocalUserID  int64  `json:"local_user_id"`
	NewAPIUserID string `json:"newapi_user_id"`
	Username     string `json:"username"`
	Email        string `json:"email,omitempty"`
	DisplayName  string `json:"display_name,omitempty"`
	IssuedAt     int64  `json:"issued_at"`
	ExpiresAt    int64  `json:"expires_at"`
}

// Manager 负责签发和解析本站用户会话
type Manager struct {
	secret     []byte
	cookieName string
	ttl        time.Duration
}

// NewManager 创建会话管理器
func NewManager(secret, cookieName string) *Manager {
	if cookieName == "" {
		cookieName = DefaultCookieName
	}

	return &Manager{
		secret:     []byte(secret),
		cookieName: cookieName,
		ttl:        defaultTTL,
	}
}

// CookieName 返回当前使用的 Cookie 名称
func (m *Manager) CookieName() string {
	return m.cookieName
}

// Issue 签发新会话
func (m *Manager) Issue(claims *Claims) (string, error) {
	if claims == nil {
		return "", fmt.Errorf("claims 不能为空")
	}
	if claims.LocalUserID <= 0 || claims.NewAPIUserID == "" || claims.Username == "" {
		return "", fmt.Errorf("claims 缺少必要字段")
	}

	now := time.Now()
	payloadClaims := *claims
	payloadClaims.IssuedAt = now.Unix()
	payloadClaims.ExpiresAt = now.Add(m.ttl).Unix()

	payload, err := json.Marshal(payloadClaims)
	if err != nil {
		return "", fmt.Errorf("序列化会话失败: %w", err)
	}

	encodedPayload := base64.RawURLEncoding.EncodeToString(payload)
	signature := m.sign(encodedPayload)
	return encodedPayload + "." + signature, nil
}

// Parse 解析并校验会话
func (m *Manager) Parse(token string) (*Claims, error) {
	parts := strings.Split(token, ".")
	if len(parts) != 2 {
		return nil, errInvalidToken
	}

	encodedPayload := parts[0]
	signature := parts[1]
	if !hmac.Equal([]byte(signature), []byte(m.sign(encodedPayload))) {
		return nil, errInvalidToken
	}

	payload, err := base64.RawURLEncoding.DecodeString(encodedPayload)
	if err != nil {
		return nil, errInvalidToken
	}

	var claims Claims
	if err := json.Unmarshal(payload, &claims); err != nil {
		return nil, errInvalidToken
	}
	if claims.ExpiresAt <= time.Now().Unix() {
		return nil, errExpiredToken
	}

	return &claims, nil
}

// BuildCookie 生成登录成功后的 Cookie
func (m *Manager) BuildCookie(token string, secure bool) *http.Cookie {
	return &http.Cookie{
		Name:     m.cookieName,
		Value:    token,
		Path:     "/",
		HttpOnly: true,
		SameSite: http.SameSiteLaxMode,
		MaxAge:   int(m.ttl.Seconds()),
		Secure:   secure,
	}
}

// BuildExpiredCookie 生成清除登录态的 Cookie
func (m *Manager) BuildExpiredCookie(secure bool) *http.Cookie {
	return &http.Cookie{
		Name:     m.cookieName,
		Value:    "",
		Path:     "/",
		HttpOnly: true,
		SameSite: http.SameSiteLaxMode,
		MaxAge:   -1,
		Secure:   secure,
		Expires:  time.Unix(0, 0),
	}
}

func (m *Manager) sign(payload string) string {
	mac := hmac.New(sha256.New, m.secret)
	mac.Write([]byte(payload))
	return base64.RawURLEncoding.EncodeToString(mac.Sum(nil))
}

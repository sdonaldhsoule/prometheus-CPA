package newapi

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"
)

// User NewAPI 当前登录用户摘要
type User struct {
	ID          string
	Username    string
	Email       string
	DisplayName string
}

// Client NewAPI 客户端
type Client struct {
	baseURL    string
	httpClient *http.Client
}

// NewClient 创建 NewAPI 客户端
func NewClient(baseURL string) *Client {
	return &Client{
		baseURL: strings.TrimSuffix(baseURL, "/"),
		httpClient: &http.Client{
			Timeout: 20 * time.Second,
		},
	}
}

// Authenticate 使用用户名密码登录，并读取当前用户信息
func (c *Client) Authenticate(ctx context.Context, username, password string) (*User, error) {
	cookies, err := c.login(ctx, username, password)
	if err != nil {
		return nil, err
	}
	if len(cookies) == 0 {
		return nil, fmt.Errorf("NewAPI 登录成功但未返回会话")
	}
	return c.getCurrentUser(ctx, cookies)
}

func (c *Client) login(ctx context.Context, username, password string) ([]*http.Cookie, error) {
	payload, err := json.Marshal(map[string]string{
		"username": username,
		"password": password,
	})
	if err != nil {
		return nil, fmt.Errorf("序列化 NewAPI 登录请求失败: %w", err)
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, c.baseURL+"/api/user/login", bytes.NewReader(payload))
	if err != nil {
		return nil, fmt.Errorf("创建 NewAPI 登录请求失败: %w", err)
	}
	req.Header.Set("Content-Type", "application/json")

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("NewAPI 登录请求失败: %w", err)
	}
	defer resp.Body.Close()

	body, _ := io.ReadAll(resp.Body)
	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("NewAPI 登录失败: %s", extractMessage(body))
	}
	if isExplicitFailure(body) {
		return nil, fmt.Errorf("NewAPI 登录失败: %s", extractMessage(body))
	}

	return resp.Cookies(), nil
}

func (c *Client) getCurrentUser(ctx context.Context, cookies []*http.Cookie) (*User, error) {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, c.baseURL+"/api/user/self", nil)
	if err != nil {
		return nil, fmt.Errorf("创建 NewAPI 用户信息请求失败: %w", err)
	}
	for _, cookie := range cookies {
		req.AddCookie(cookie)
	}

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("读取 NewAPI 当前用户失败: %w", err)
	}
	defer resp.Body.Close()

	body, _ := io.ReadAll(resp.Body)
	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("读取 NewAPI 当前用户失败: %s", extractMessage(body))
	}
	if isExplicitFailure(body) {
		return nil, fmt.Errorf("读取 NewAPI 当前用户失败: %s", extractMessage(body))
	}

	user, err := parseUser(body)
	if err != nil {
		return nil, err
	}
	return user, nil
}

func parseUser(body []byte) (*User, error) {
	payload := body

	var envelope map[string]json.RawMessage
	if err := json.Unmarshal(body, &envelope); err == nil {
		for _, key := range []string{"data", "user"} {
			if raw, ok := envelope[key]; ok && len(raw) > 0 && string(raw) != "null" {
				payload = raw
				break
			}
		}
	}

	var candidate map[string]any
	if err := json.Unmarshal(payload, &candidate); err != nil {
		return nil, fmt.Errorf("解析 NewAPI 当前用户失败")
	}

	user := &User{
		ID:          firstNonEmpty(toString(candidate["id"]), toString(candidate["user_id"])),
		Username:    firstNonEmpty(toString(candidate["username"]), toString(candidate["name"]), toString(candidate["email"])),
		Email:       toString(candidate["email"]),
		DisplayName: firstNonEmpty(toString(candidate["display_name"]), toString(candidate["displayName"]), toString(candidate["nickname"])),
	}

	if user.ID == "" || user.Username == "" {
		return nil, fmt.Errorf("NewAPI 当前用户信息缺少必要字段")
	}

	return user, nil
}

func isExplicitFailure(body []byte) bool {
	var result map[string]any
	if err := json.Unmarshal(body, &result); err != nil {
		return false
	}
	if success, ok := result["success"].(bool); ok {
		return !success
	}
	return false
}

func extractMessage(body []byte) string {
	var result map[string]any
	if err := json.Unmarshal(body, &result); err != nil {
		text := strings.TrimSpace(string(body))
		if text == "" {
			return "未知错误"
		}
		return text
	}

	for _, key := range []string{"message", "msg", "error"} {
		if value := strings.TrimSpace(toString(result[key])); value != "" {
			return value
		}
	}
	return "未知错误"
}

func firstNonEmpty(values ...string) string {
	for _, value := range values {
		if strings.TrimSpace(value) != "" {
			return value
		}
	}
	return ""
}

func toString(value any) string {
	switch v := value.(type) {
	case nil:
		return ""
	case string:
		return v
	case float64:
		return strings.TrimSuffix(strings.TrimSuffix(fmt.Sprintf("%.0f", v), ".0"), ".")
	case json.Number:
		return v.String()
	default:
		return fmt.Sprint(v)
	}
}

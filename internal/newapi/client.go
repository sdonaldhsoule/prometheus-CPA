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

type loginResult struct {
	cookies []*http.Cookie
	user    *User
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
	result, err := c.login(ctx, username, password)
	if err != nil {
		return nil, err
	}
	if len(result.cookies) == 0 {
		return nil, fmt.Errorf("NewAPI 登录成功但未返回会话")
	}

	if result.user != nil && result.user.ID != "" {
		user, err := c.getCurrentUser(ctx, result.cookies, result.user.ID)
		if err == nil {
			return mergeUsers(result.user, user), nil
		}
		if result.user.Username != "" {
			return result.user, nil
		}
		return nil, err
	}

	return c.getCurrentUser(ctx, result.cookies, "")
}

func (c *Client) login(ctx context.Context, username, password string) (*loginResult, error) {
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

	return &loginResult{
		cookies: resp.Cookies(),
		user:    extractUser(body),
	}, nil
}

func (c *Client) getCurrentUser(ctx context.Context, cookies []*http.Cookie, userID string) (*User, error) {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, c.baseURL+"/api/user/self", nil)
	if err != nil {
		return nil, fmt.Errorf("创建 NewAPI 用户信息请求失败: %w", err)
	}
	for _, cookie := range cookies {
		req.AddCookie(cookie)
	}
	if strings.TrimSpace(userID) != "" {
		req.Header.Set("New-Api-User", userID)
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
	candidate, err := extractUserMap(body)
	if err != nil {
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

func extractUser(body []byte) *User {
	candidate, err := extractUserMap(body)
	if err != nil {
		return nil
	}

	user := &User{
		ID:          firstNonEmpty(toString(candidate["id"]), toString(candidate["user_id"])),
		Username:    firstNonEmpty(toString(candidate["username"]), toString(candidate["name"]), toString(candidate["email"])),
		Email:       toString(candidate["email"]),
		DisplayName: firstNonEmpty(toString(candidate["display_name"]), toString(candidate["displayName"]), toString(candidate["nickname"])),
	}

	if user.ID == "" && user.Username == "" && user.Email == "" && user.DisplayName == "" {
		return nil
	}

	return user
}

func extractUserMap(body []byte) (map[string]any, error) {
	payloads := candidatePayloads(body)
	for _, payload := range payloads {
		var candidate map[string]any
		if err := json.Unmarshal(payload, &candidate); err != nil {
			continue
		}
		if nestedUser, ok := candidate["user"]; ok {
			if userMap, ok := nestedUser.(map[string]any); ok {
				return userMap, nil
			}
		}
		if nestedData, ok := candidate["data"]; ok {
			if dataMap, ok := nestedData.(map[string]any); ok {
				if nestedUser, ok := dataMap["user"]; ok {
					if userMap, ok := nestedUser.(map[string]any); ok {
						return userMap, nil
					}
				}
				if looksLikeUserMap(dataMap) {
					return dataMap, nil
				}
			}
		}
		if looksLikeUserMap(candidate) {
			return candidate, nil
		}
	}
	return nil, fmt.Errorf("no user candidate")
}

func candidatePayloads(body []byte) [][]byte {
	payloads := [][]byte{body}

	var envelope map[string]json.RawMessage
	if err := json.Unmarshal(body, &envelope); err != nil {
		return payloads
	}

	for _, key := range []string{"data", "user"} {
		if raw, ok := envelope[key]; ok && len(raw) > 0 && string(raw) != "null" {
			payloads = append(payloads, raw)
		}
	}

	return payloads
}

func mergeUsers(preferred, fallback *User) *User {
	if preferred == nil {
		return fallback
	}
	if fallback == nil {
		return preferred
	}

	merged := *fallback
	if strings.TrimSpace(merged.ID) == "" {
		merged.ID = preferred.ID
	}
	if strings.TrimSpace(merged.Username) == "" {
		merged.Username = preferred.Username
	}
	if strings.TrimSpace(merged.Email) == "" {
		merged.Email = preferred.Email
	}
	if strings.TrimSpace(merged.DisplayName) == "" {
		merged.DisplayName = preferred.DisplayName
	}
	return &merged
}

func looksLikeUserMap(candidate map[string]any) bool {
	for _, key := range []string{"id", "user_id", "username", "email", "display_name", "displayName", "nickname", "name"} {
		if value := strings.TrimSpace(toString(candidate[key])); value != "" {
			return true
		}
	}
	return false
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

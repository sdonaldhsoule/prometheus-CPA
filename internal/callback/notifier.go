package callback

import (
	"bytes"
	"context"
	"crypto/hmac"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"time"
)

// CallbackData 回调数据
type CallbackData struct {
	CredentialID   int64  `json:"credential_id"`
	CredentialType string `json:"credential_type"`
	Email          string `json:"email"`
	ProjectID      string `json:"project_id"`
	CDKCode        string `json:"cdk_code"`
	Timestamp      int64  `json:"timestamp"`
	Signature      string `json:"signature"`
}

// CallbackResult 回调结果
type CallbackResult struct {
	Success      bool   `json:"success"`
	StatusCode   int    `json:"status_code"`
	ResponseBody string `json:"response_body"`
	Error        string `json:"error,omitempty"`
}

// Notifier 回调通知器
type Notifier struct {
	callbackURL string
	secret      string
	httpClient  *http.Client
}

// NewNotifier 创建回调通知器
func NewNotifier(callbackURL, secret string) *Notifier {
	return &Notifier{
		callbackURL: callbackURL,
		secret:      secret,
		httpClient: &http.Client{
			Timeout: 30 * time.Second,
		},
	}
}

// Notify 发送回调通知
func (n *Notifier) Notify(ctx context.Context, data *CallbackData) (*CallbackResult, error) {
	if n.callbackURL == "" {
		return &CallbackResult{Success: true, StatusCode: 200, ResponseBody: "no callback configured"}, nil
	}

	// 设置时间戳
	data.Timestamp = time.Now().Unix()

	// 生成签名
	data.Signature = n.generateSignature(data)

	// 序列化数据
	body, err := json.Marshal(data)
	if err != nil {
		return nil, fmt.Errorf("marshal callback data: %w", err)
	}

	// 创建请求
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, n.callbackURL, bytes.NewReader(body))
	if err != nil {
		return nil, fmt.Errorf("create request: %w", err)
	}

	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("X-Signature", data.Signature)
	req.Header.Set("X-Timestamp", fmt.Sprintf("%d", data.Timestamp))

	// 发送请求
	resp, err := n.httpClient.Do(req)
	if err != nil {
		return &CallbackResult{
			Success: false,
			Error:   fmt.Sprintf("request failed: %v", err),
		}, nil
	}
	defer resp.Body.Close()

	// 读取响应
	respBody, _ := io.ReadAll(io.LimitReader(resp.Body, 4096))

	result := &CallbackResult{
		Success:      resp.StatusCode >= 200 && resp.StatusCode < 300,
		StatusCode:   resp.StatusCode,
		ResponseBody: string(respBody),
	}

	if !result.Success {
		result.Error = fmt.Sprintf("callback failed with status %d", resp.StatusCode)
	}

	return result, nil
}

// generateSignature 生成HMAC签名
func (n *Notifier) generateSignature(data *CallbackData) string {
	// 构建签名字符串
	signStr := fmt.Sprintf("%d:%s:%s:%s:%d",
		data.CredentialID,
		data.Email,
		data.ProjectID,
		data.CDKCode,
		data.Timestamp,
	)

	h := hmac.New(sha256.New, []byte(n.secret))
	h.Write([]byte(signStr))
	return hex.EncodeToString(h.Sum(nil))
}

// VerifySignature 验证签名 (供接收方使用)
func VerifySignature(data *CallbackData, secret string) bool {
	signStr := fmt.Sprintf("%d:%s:%s:%s:%d",
		data.CredentialID,
		data.Email,
		data.ProjectID,
		data.CDKCode,
		data.Timestamp,
	)

	h := hmac.New(sha256.New, []byte(secret))
	h.Write([]byte(signStr))
	expected := hex.EncodeToString(h.Sum(nil))

	return hmac.Equal([]byte(expected), []byte(data.Signature))
}

package turnstile

import (
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"net/url"
	"time"
)

// Client Cloudflare Turnstile 验证客户端
type Client struct {
	secretKey string
	verifyURL string
	http      *http.Client
}

// NewClient 创建 Turnstile 客户端
func NewClient(secretKey, verifyURL string) *Client {
	if verifyURL == "" {
		verifyURL = "https://challenges.cloudflare.com/turnstile/v0/siteverify"
	}
	return &Client{
		secretKey: secretKey,
		verifyURL: verifyURL,
		http: &http.Client{
			Timeout: 5 * time.Second,
		},
	}
}

// VerifyResponse 验证响应
type VerifyResponse struct {
	Success     bool      `json:"success"`
	ChallengeTS time.Time `json:"challenge_ts"`
	Hostname    string    `json:"hostname"`
	ErrorCodes  []string  `json:"error-codes,omitempty"`
	Action      string    `json:"action,omitempty"`
	CData       string    `json:"cdata,omitempty"`
}

// Verify 校验 token
func (c *Client) Verify(token, remoteIP string) (*VerifyResponse, error) {
	if c.secretKey == "" {
		return nil, errors.New("Turnstile secret key 未配置")
	}
	if token == "" {
		return nil, errors.New("Turnstile token 为空")
	}

	form := url.Values{}
	form.Set("secret", c.secretKey)
	form.Set("response", token)
	if remoteIP != "" {
		form.Set("remoteip", remoteIP)
	}

	resp, err := c.http.PostForm(c.verifyURL, form)
	if err != nil {
		return nil, fmt.Errorf("Turnstile 验证请求失败: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("Turnstile 验证返回状态码: %d", resp.StatusCode)
	}

	var result VerifyResponse
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return nil, fmt.Errorf("解析 Turnstile 响应失败: %w", err)
	}

	return &result, nil
}

// IsSkipped 当 secret key 未配置时跳过校验（开发环境用）
func (c *Client) IsSkipped() bool {
	return c.secretKey == ""
}
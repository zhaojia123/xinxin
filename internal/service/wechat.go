package service

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"time"

	"friends-records/internal/apperror"
)

type WeChatSession struct{ OpenID, UnionID string }
type CodeExchanger interface {
	Exchange(context.Context, string) (WeChatSession, error)
}
type WeChatClient struct {
	AppID, AppSecret string
	Client           *http.Client
}

func NewWeChatClient(appID, secret string) *WeChatClient {
	return &WeChatClient{appID, secret, &http.Client{Timeout: 8 * time.Second}}
}
func (c *WeChatClient) Exchange(ctx context.Context, code string) (WeChatSession, error) {
	query := url.Values{"appid": {c.AppID}, "secret": {c.AppSecret}, "js_code": {code}, "grant_type": {"authorization_code"}}
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, "https://api.weixin.qq.com/sns/jscode2session?"+query.Encode(), nil)
	if err != nil {
		return WeChatSession{}, apperror.Wrap(err, "创建微信登录请求失败")
	}
	resp, err := c.Client.Do(req)
	if err != nil {
		return WeChatSession{}, apperror.Wrap(err, "调用微信登录接口失败")
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		return WeChatSession{}, apperror.New(fmt.Sprintf("微信登录接口返回HTTP %d", resp.StatusCode))
	}
	var data struct {
		OpenID  string `json:"openid"`
		UnionID string `json:"unionid"`
		ErrCode int    `json:"errcode"`
		ErrMsg  string `json:"errmsg"`
	}
	if err := json.NewDecoder(io.LimitReader(resp.Body, 1<<20)).Decode(&data); err != nil {
		return WeChatSession{}, apperror.Wrap(err, "解析微信登录响应失败")
	}
	if data.ErrCode != 0 {
		return WeChatSession{}, apperror.New(fmt.Sprintf("微信登录失败，错误码%d：%s", data.ErrCode, data.ErrMsg))
	}
	if data.OpenID == "" {
		return WeChatSession{}, apperror.New("微信登录响应中没有openid")
	}
	return WeChatSession{data.OpenID, data.UnionID}, nil
}

type MockWeChatClient struct{}

func (MockWeChatClient) Exchange(_ context.Context, code string) (WeChatSession, error) {
	if code == "" {
		return WeChatSession{}, apperror.New("微信登录code不能为空")
	}
	// 本地模拟用户保持固定，避免 wx.login 每次换 code 时重复创建账号。
	sum := sha256.Sum256([]byte("friends-records:local-developer"))
	return WeChatSession{OpenID: "mock_" + hex.EncodeToString(sum[:12])}, nil
}

package service

import (
	"context"
	"strings"
	"time"

	"friends-records/internal/apperror"
	"friends-records/internal/models/mysql"
	"friends-records/internal/token"
)

var ErrMiniUserDisabled = apperror.New("小程序用户尚未启用")

type Mini struct {
	Store        *mysql.Store
	Exchanger    CodeExchanger
	Tokens       *token.Manager
	AutoRegister bool
}
type MiniLoginResult struct {
	Token string         `json:"token"`
	User  mysql.MiniUser `json:"user"`
}

func (s Mini) Login(ctx context.Context, code string) (MiniLoginResult, error) {
	code = strings.TrimSpace(code)
	if code == "" || len(code) > 256 {
		return MiniLoginResult{}, apperror.New("微信登录code为空或长度不正确")
	}
	session, err := s.Exchanger.Exchange(ctx, code)
	if err != nil {
		return MiniLoginResult{}, err
	}
	user, err := s.Store.MiniLogin(ctx, session.OpenID, session.UnionID, s.AutoRegister)
	if err != nil {
		return MiniLoginResult{}, err
	}
	if !user.Enabled {
		return MiniLoginResult{}, ErrMiniUserDisabled
	}
	value, err := s.Tokens.Issue("mini", user.ID, 7*24*time.Hour)
	if err != nil {
		return MiniLoginResult{}, apperror.Wrap(err, "生成小程序登录令牌失败")
	}
	return MiniLoginResult{value, user}, nil
}

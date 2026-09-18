package request

import (
	"fmt"
	"regexp"
	"strings"
)

var miniModuleKeyPattern = regexp.MustCompile(`^[a-z0-9][a-z0-9_-]{0,63}$`)

// MiniUserPermissionInput 是后台分配小程序用户权限的入参。
type MiniUserPermissionInput struct {
	DisplayName string                      `json:"display_name"`
	Enabled     bool                        `json:"enabled"`
	Permissions map[string]MiniModuleAccess `json:"permissions"`
}

type MiniModuleAccess struct {
	View   bool `json:"view"`
	Create bool `json:"create"`
	Edit   bool `json:"edit"`
	Delete bool `json:"delete"`
}

func (v MiniUserPermissionInput) Validate() error {
	if len([]rune(strings.TrimSpace(v.DisplayName))) > 64 {
		return fmt.Errorf("用户名称不能超过64个字符")
	}
	if len(v.Permissions) > 100 {
		return fmt.Errorf("模块权限数量不能超过100个")
	}
	seen := map[string]struct{}{}
	for key := range v.Permissions {
		if !miniModuleKeyPattern.MatchString(key) {
			return fmt.Errorf("模块标识格式不正确：%s", key)
		}
		seen[key] = struct{}{}
	}
	return nil
}

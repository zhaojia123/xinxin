package mysql

import (
	"context"
	"database/sql"
	"errors"

	"friends-records/internal/apperror"
)

var ErrInvalidCredentials = apperror.New("用户名或密码错误")

type AdminUser struct {
	ID                                  uint64
	Username, PasswordHash, DisplayName string
	Enabled                             bool
}
type MiniUser struct {
	ID           uint64                      `json:"id"`
	DisplayName  string                      `json:"display_name"`
	Enabled      bool                        `json:"enabled"`
	CanLedger    bool                        `json:"can_ledger"`
	CanPurchases bool                        `json:"can_purchases"`
	CanEmployees bool                        `json:"can_employees"`
	Permissions  []string                    `json:"permissions"`
	Actions      map[string]MiniActionAccess `json:"actions"`
}

type MiniActionAccess struct {
	View   bool `json:"view"`
	Create bool `json:"create"`
	Edit   bool `json:"edit"`
	Delete bool `json:"delete"`
}

func (s *Store) AdminByUsername(ctx context.Context, username string) (AdminUser, error) {
	if err := s.ready(); err != nil {
		return AdminUser{}, err
	}
	var user AdminUser
	err := s.DB.QueryRowContext(ctx, `SELECT id,username,password_hash,display_name,enabled FROM admin_users WHERE username=? LIMIT 1`, username).Scan(&user.ID, &user.Username, &user.PasswordHash, &user.DisplayName, &user.Enabled)
	if errors.Is(err, sql.ErrNoRows) {
		return AdminUser{}, ErrInvalidCredentials
	}
	if err != nil {
		return AdminUser{}, apperror.Wrap(err, "查询后台用户失败")
	}
	return user, nil
}

func (s *Store) MiniLogin(ctx context.Context, openID, unionID string, autoRegister bool) (MiniUser, error) {
	if err := s.ready(); err != nil {
		return MiniUser{}, err
	}
	if _, err := s.DB.ExecContext(ctx, `INSERT INTO mini_users(openid,unionid,enabled,last_login_at) VALUES(?,?,?,NOW()) ON DUPLICATE KEY UPDATE unionid=IF(VALUES(unionid)='',unionid,VALUES(unionid)),last_login_at=NOW()`, openID, unionID, autoRegister); err != nil {
		return MiniUser{}, apperror.Wrap(err, "创建或更新小程序用户失败")
	}
	var user MiniUser
	if err := s.DB.QueryRowContext(ctx, `SELECT id,display_name,enabled,can_ledger,can_purchases,can_employees FROM mini_users WHERE openid=? LIMIT 1`, openID).Scan(&user.ID, &user.DisplayName, &user.Enabled, &user.CanLedger, &user.CanPurchases, &user.CanEmployees); err != nil {
		return MiniUser{}, apperror.Wrap(err, "读取小程序用户失败")
	}
	// 新用户沿用当前三个模块的默认权限；后续新增模块只需向权限表增加记录。
	if _, err := s.DB.ExecContext(ctx, `INSERT IGNORE INTO mini_user_permissions (mini_user_id,module_key,enabled) VALUES (?,?,?),(?,?,?),(?,?,?)`, user.ID, "ledger", user.CanLedger, user.ID, "purchases", user.CanPurchases, user.ID, "employees", user.CanEmployees); err != nil {
		return MiniUser{}, apperror.Wrap(err, "初始化小程序用户权限失败")
	}
	permissions, err := s.MiniPermissions(ctx, user.ID)
	if err != nil {
		return MiniUser{}, apperror.Wrap(err, "读取小程序用户权限失败")
	}
	user.Permissions = permissions
	user.Actions, err = s.MiniActionPermissions(ctx, user.ID)
	if err != nil {
		return MiniUser{}, apperror.Wrap(err, "读取小程序操作权限失败")
	}
	return user, nil
}

package mysql

import (
	"context"
	"strings"

	"friends-records/api/request"
)

type AdminMiniUser struct {
	ID          uint64                              `json:"id"`
	OpenID      string                              `json:"openid"`
	DisplayName string                              `json:"display_name"`
	Enabled     bool                                `json:"enabled"`
	Permissions map[string]request.MiniModuleAccess `json:"permissions"`
}

type MiniModule struct {
	Key  string `json:"key"`
	Name string `json:"name"`
}

// AdminMiniUsers 返回后台用于分配权限的小程序用户列表。
func (s *Store) AdminMiniUsers(ctx context.Context) ([]AdminMiniUser, []MiniModule, error) {
	if err := s.ready(); err != nil {
		return nil, nil, err
	}
	rows, err := s.DB.QueryContext(ctx, `SELECT id,openid,display_name,enabled FROM mini_users ORDER BY id DESC`)
	if err != nil {
		return nil, nil, err
	}
	defer rows.Close()
	users := []AdminMiniUser{}
	for rows.Next() {
		var user AdminMiniUser
		if err := rows.Scan(&user.ID, &user.OpenID, &user.DisplayName, &user.Enabled); err != nil {
			return nil, nil, err
		}
		user.Permissions = map[string]request.MiniModuleAccess{}
		permissionRows, err := s.DB.QueryContext(ctx, `SELECT module_key,enabled,can_view,can_create,can_edit,can_delete FROM mini_user_permissions WHERE mini_user_id=?`, user.ID)
		if err != nil {
			return nil, nil, err
		}
		for permissionRows.Next() {
			var key string
			var enabled bool
			var access request.MiniModuleAccess
			if err := permissionRows.Scan(&key, &enabled, &access.View, &access.Create, &access.Edit, &access.Delete); err != nil {
				permissionRows.Close()
				return nil, nil, err
			}
			if !enabled {
				access = request.MiniModuleAccess{}
			}
			user.Permissions[key] = access
		}
		if err := permissionRows.Err(); err != nil {
			permissionRows.Close()
			return nil, nil, err
		}
		permissionRows.Close()
		users = append(users, user)
	}
	if err := rows.Err(); err != nil {
		return nil, nil, err
	}
	moduleRows, err := s.DB.QueryContext(ctx, `SELECT DISTINCT module_key FROM mini_user_permissions ORDER BY module_key`)
	if err != nil {
		return nil, nil, err
	}
	defer moduleRows.Close()
	keys := map[string]bool{}
	modules := []MiniModule{}
	for moduleRows.Next() {
		var key string
		if err := moduleRows.Scan(&key); err != nil {
			return nil, nil, err
		}
		keys[key] = true
		modules = append(modules, MiniModule{Key: key, Name: miniModuleName(key)})
	}
	if err := moduleRows.Err(); err != nil {
		return nil, nil, err
	}
	for _, module := range []MiniModule{{Key: "ledger", Name: "台账"}, {Key: "purchases", Name: "采购"}, {Key: "employees", Name: "员工管理"}} {
		if !keys[module.Key] {
			modules = append(modules, module)
		}
	}
	return users, modules, nil
}

func miniModuleName(key string) string {
	switch key {
	case "ledger":
		return "台账"
	case "purchases":
		return "采购"
	case "employees":
		return "员工管理"
	default:
		return key
	}
}

// UpdateMiniUserPermissions 使用启用字段做权限控制，不物理删除权限记录。
func (s *Store) UpdateMiniUserPermissions(ctx context.Context, id uint64, input request.MiniUserPermissionInput) error {
	if err := s.ready(); err != nil {
		return err
	}
	tx, err := s.DB.BeginTx(ctx, nil)
	if err != nil {
		return err
	}
	defer tx.Rollback()
	result, err := tx.ExecContext(ctx, `UPDATE mini_users SET display_name=?,enabled=?,updated_at=NOW() WHERE id=?`, strings.TrimSpace(input.DisplayName), input.Enabled, id)
	if err != nil {
		return err
	}
	if affected, _ := result.RowsAffected(); affected == 0 {
		return ErrNotFound
	}
	if _, err := tx.ExecContext(ctx, `UPDATE mini_user_permissions SET enabled=0,can_view=0,can_create=0,can_edit=0,can_delete=0,updated_at=NOW() WHERE mini_user_id=?`, id); err != nil {
		return err
	}
	for key, access := range input.Permissions {
		enabled := access.View || access.Create || access.Edit || access.Delete
		if _, err := tx.ExecContext(ctx, `INSERT INTO mini_user_permissions (mini_user_id,module_key,enabled,can_view,can_create,can_edit,can_delete) VALUES (?,?,?,?,?,?,?) ON DUPLICATE KEY UPDATE enabled=VALUES(enabled),can_view=VALUES(can_view),can_create=VALUES(can_create),can_edit=VALUES(can_edit),can_delete=VALUES(can_delete),updated_at=NOW()`, id, key, enabled, access.View, access.Create, access.Edit, access.Delete); err != nil {
			return err
		}
	}
	return tx.Commit()
}

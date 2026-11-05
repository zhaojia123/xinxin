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

// miniProgramModules 是当前小程序对外开放的最小业务范围。
// 其他经营模块只在后台维护，避免小程序权限误开后访问不必要的数据。
var miniProgramModules = []MiniModule{
	{Key: "employees", Name: "员工档案管理"},
	{Key: "attendance", Name: "请假与调休"},
}

func isMiniProgramModule(key string) bool {
	for _, module := range miniProgramModules {
		if module.Key == key {
			return true
		}
	}
	return false
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
		permissionRows, err := s.DB.QueryContext(ctx, `SELECT module_key,enabled,can_view,can_create,can_edit,can_delete FROM mini_user_permissions WHERE mini_user_id=? AND module_key IN ('employees','attendance')`, user.ID)
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
	return users, append([]MiniModule(nil), miniProgramModules...), nil
}

func miniModuleName(key string) string {
	switch key {
	case "ledger":
		return "台账"
	case "purchases":
		return "采购"
	case "employees":
		return "员工档案管理"
	case "attendance":
		return "请假管理"
	case "work_logs":
		return "临时工工时"
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
	permissions := make(map[string]request.MiniModuleAccess, len(input.Permissions))
	for key, access := range input.Permissions {
		if !isMiniProgramModule(key) {
			continue
		}
		// 任何写权限都必须带查看权限，否则页面无法读取已有数据。
		if access.Create || access.Edit || access.Delete {
			access.View = true
		}
		permissions[key] = access
	}
	// 请假和调休选择员工时必须读取员工名称；只分配请假权限时自动附带员工查看权限。
	if attendance := permissions["attendance"]; attendance.View || attendance.Create || attendance.Edit || attendance.Delete {
		employee := permissions["employees"]
		employee.View = true
		permissions["employees"] = employee
	}
	for key, access := range permissions {
		enabled := access.View || access.Create || access.Edit || access.Delete
		if _, err := tx.ExecContext(ctx, `INSERT INTO mini_user_permissions (mini_user_id,module_key,enabled,can_view,can_create,can_edit,can_delete) VALUES (?,?,?,?,?,?,?) ON DUPLICATE KEY UPDATE enabled=VALUES(enabled),can_view=VALUES(can_view),can_create=VALUES(can_create),can_edit=VALUES(can_edit),can_delete=VALUES(can_delete),updated_at=NOW()`, id, key, enabled, access.View, access.Create, access.Edit, access.Delete); err != nil {
			return err
		}
	}
	return tx.Commit()
}

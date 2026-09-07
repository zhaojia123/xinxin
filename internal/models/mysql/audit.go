package mysql

import (
	"context"
	"friends-records/api/response"
	"friends-records/internal/apperror"
)

func (s *Store) AuditLogs(ctx context.Context) ([]response.AuditLog, error) {
	if err := s.ready(); err != nil {
		return nil, err
	}
	rows, err := s.DB.QueryContext(ctx, `SELECT l.id,DATE_FORMAT(l.created_at,'%Y-%m-%d %H:%i:%s'),COALESCE(u.display_name,'系统'),l.module,l.action,l.target_text,l.ip_address,l.result FROM audit_logs l LEFT JOIN admin_users u ON u.id=l.admin_user_id ORDER BY l.created_at DESC,l.id DESC LIMIT 500`)
	if err != nil {
		return nil, apperror.Wrap(err, "查询操作日志失败")
	}
	defer rows.Close()
	result := make([]response.AuditLog, 0)
	for rows.Next() {
		var v response.AuditLog
		var state string
		if err := rows.Scan(&v.ID, &v.Time, &v.Operator, &v.Module, &v.Action, &v.Target, &v.IP, &state); err != nil {
			return nil, apperror.Wrap(err, "读取操作日志失败")
		}
		if state == "success" {
			v.Result, v.ResultClass = "成功", "active"
		} else {
			v.Result, v.ResultClass = "失败", "inactive"
		}
		result = append(result, v)
	}
	if err := rows.Err(); err != nil {
		return nil, apperror.Wrap(err, "遍历操作日志失败")
	}
	return result, nil
}

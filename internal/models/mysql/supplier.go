package mysql

import (
	"context"
	"database/sql"

	"friends-records/api/request"
	"friends-records/api/response"
	"friends-records/internal/apperror"
)

// supplierDisplayOrder 与供货商对账单的既定顺序一致；未配置的新供货商排在末尾。
const supplierDisplayOrder = `CASE name
	WHEN '菜卜卜' THEN 1
	WHEN '阿淘水产' THEN 2
	WHEN '好享尝' THEN 3
	WHEN '群利鑫（调味品）' THEN 4
	WHEN '秀源鑫欢猪肉店' THEN 5
	WHEN '凌慢慢鲜面' THEN 6
	WHEN '宸盛杰' THEN 7
	WHEN '速达宇' THEN 8
	WHEN '豫集翔（易耗品）' THEN 9
	WHEN '米油' THEN 10
	ELSE 999
END`

// Suppliers 返回仍可用于录入对账的供货商。
func (s *Store) Suppliers(ctx context.Context) ([]response.Supplier, error) {
	if err := s.ready(); err != nil {
		return nil, err
	}
	rows, err := s.DB.QueryContext(ctx, `SELECT id,name,contact_name,mobile,remark FROM supplier_accounts WHERE active=1 ORDER BY `+supplierDisplayOrder+`,name,id`)
	if err != nil {
		return nil, apperror.Wrap(err, "查询供货商失败")
	}
	defer rows.Close()
	items := make([]response.Supplier, 0)
	for rows.Next() {
		var item response.Supplier
		if err := rows.Scan(&item.ID, &item.Name, &item.Contact, &item.Mobile, &item.Remark); err != nil {
			return nil, apperror.Wrap(err, "读取供货商失败")
		}
		item.Seq = len(items) + 1
		items = append(items, item)
	}
	return items, rows.Err()
}

// SaveSupplier 保存供货商；改名时同步历史档口对账的展示名称。
func (s *Store) SaveSupplier(ctx context.Context, id, actor uint64, input request.SupplierInput) (uint64, error) {
	if err := s.ready(); err != nil {
		return 0, err
	}
	tx, err := s.DB.BeginTx(ctx, nil)
	if err != nil {
		return 0, apperror.Wrap(err, "开始保存供货商事务失败")
	}
	defer tx.Rollback()
	if id == 0 {
		// 同名但已停用的供货商直接恢复，避免历史名称占用导致无法重新启用。
		result, err := tx.ExecContext(ctx, `INSERT INTO supplier_accounts(name,contact_name,mobile,remark,active) VALUES(?,?,?,?,1) ON DUPLICATE KEY UPDATE contact_name=VALUES(contact_name),mobile=VALUES(mobile),remark=VALUES(remark),active=1,id=LAST_INSERT_ID(id)`, input.Name, input.Contact, input.Mobile, input.Remark)
		if err != nil {
			return 0, apperror.Wrap(err, "新增供货商失败")
		}
		last, err := result.LastInsertId()
		if err != nil {
			return 0, err
		}
		id = uint64(last)
	} else {
		var oldName string
		if err := tx.QueryRowContext(ctx, `SELECT name FROM supplier_accounts WHERE id=? AND active=1 FOR UPDATE`, id).Scan(&oldName); err == sql.ErrNoRows {
			return 0, ErrNotFound
		} else if err != nil {
			return 0, err
		}
		if _, err := tx.ExecContext(ctx, `UPDATE supplier_accounts SET name=?,contact_name=?,mobile=?,remark=? WHERE id=?`, input.Name, input.Contact, input.Mobile, input.Remark, id); err != nil {
			return 0, apperror.Wrap(err, "修改供货商失败")
		}
		if oldName != input.Name {
			if _, err := tx.ExecContext(ctx, `UPDATE stall_cost_entries SET supplier_name=? WHERE supplier_name=?`, input.Name, oldName); err != nil {
				return 0, apperror.Wrap(err, "同步供货商名称失败")
			}
		}
	}
	if err := miniAudit(ctx, tx, actor, "suppliers", id, "保存供货商："+input.Name); err != nil {
		return 0, err
	}
	return id, tx.Commit()
}

func (s *Store) DeleteSupplier(ctx context.Context, id, actor uint64) error {
	if err := s.ready(); err != nil {
		return err
	}
	tx, err := s.DB.BeginTx(ctx, nil)
	if err != nil {
		return err
	}
	defer tx.Rollback()
	var name string
	if err := tx.QueryRowContext(ctx, `SELECT name FROM supplier_accounts WHERE id=? AND active=1 FOR UPDATE`, id).Scan(&name); err == sql.ErrNoRows {
		return ErrNotFound
	} else if err != nil {
		return err
	}
	if _, err := tx.ExecContext(ctx, `UPDATE supplier_accounts SET active=0 WHERE id=?`, id); err != nil {
		return apperror.Wrap(err, "删除供货商失败")
	}
	if err := miniAudit(ctx, tx, actor, "suppliers", id, "删除供货商："+name); err != nil {
		return err
	}
	return tx.Commit()
}

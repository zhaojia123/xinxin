package mysql

import (
	"context"
	"database/sql"
	"fmt"
	"math"
	"strconv"
	"strings"

	"friends-records/api/request"
	"friends-records/api/response"
	"friends-records/internal/apperror"
)

func purchaseTotal(quantity, unitPrice string) (string, error) {
	quantityValue, err := strconv.ParseFloat(quantity, 64)
	if err != nil {
		return "", MiniInputError{"采购数量格式不正确"}
	}
	priceValue, err := strconv.ParseFloat(unitPrice, 64)
	if err != nil {
		return "", MiniInputError{"单价格式不正确"}
	}
	return fmt.Sprintf("%.2f", math.Round(quantityValue*priceValue*100)/100), nil
}

type purchaseSnapshot struct {
	Date, SupplierName, Category, ProductName, Quantity, Unit, UnitPrice, TotalPrice, Remark string
}

// purchaseChangeDetail 生成最近一次编辑的前后值，便于列表快速核对采购变更。
func purchaseChangeDetail(old purchaseSnapshot, next request.PurchaseInput, total string) string {
	values := []struct {
		label, before, after string
		numeric              bool
	}{
		{"日期", old.Date, next.Date, false}, {"供货商", old.SupplierName, next.SupplierName, false},
		{"品类", old.Category, next.Category, false}, {"菜品/物料", old.ProductName, next.ProductName, false},
		{"数量", old.Quantity, next.Quantity, true}, {"单位", old.Unit, next.Unit, false},
		{"单价", old.UnitPrice, next.UnitPrice, true}, {"总价", old.TotalPrice, total, true},
		{"备注", old.Remark, next.Remark, false},
	}
	changes := make([]string, 0, len(values))
	for _, value := range values {
		if value.numeric {
			before, beforeErr := strconv.ParseFloat(value.before, 64)
			after, afterErr := strconv.ParseFloat(value.after, 64)
			if beforeErr == nil && afterErr == nil && math.Abs(before-after) < 0.000001 {
				continue
			}
		} else if value.before == value.after {
			continue
		}
		beforeText, afterText := value.before, value.after
		if beforeText == "" {
			beforeText = "（空）"
		}
		if afterText == "" {
			afterText = "（空）"
		}
		changes = append(changes, fmt.Sprintf("%s：%s → %s", value.label, beforeText, afterText))
	}
	return strings.Join(changes, "；")
}

func (s *Store) PurchaseItems(ctx context.Context, start, end string) ([]response.PurchaseItem, error) {
	if err := s.ready(); err != nil {
		return nil, err
	}
	rows, err := s.DB.QueryContext(ctx, `SELECT id,DATE_FORMAT(purchase_date,'%Y-%m-%d'),supplier_name,category,product_name,CAST(quantity AS CHAR),unit,CAST(unit_price AS CHAR),CAST(total_price AS CHAR),remark,last_change_detail,DATE_FORMAT(last_changed_at,'%Y-%m-%d %H:%i:%s'),last_change_detail IS NOT NULL AND last_change_detail<>'' FROM purchase_items WHERE active=1 AND purchase_date>=? AND purchase_date<DATE_ADD(?,INTERVAL 1 DAY) ORDER BY purchase_date,category,id`, start, end)
	if err != nil {
		return nil, apperror.Wrap(err, "查询采购明细失败")
	}
	defer rows.Close()
	items := make([]response.PurchaseItem, 0)
	for rows.Next() {
		var item response.PurchaseItem
		var changeDetail, changedAt sql.NullString
		if err := rows.Scan(&item.ID, &item.Date, &item.SupplierName, &item.Category, &item.ProductName, &item.Quantity, &item.Unit, &item.UnitPrice, &item.TotalPrice, &item.Remark, &changeDetail, &changedAt, &item.Modified); err != nil {
			return nil, apperror.Wrap(err, "读取采购明细失败")
		}
		item.ChangeDetail, item.ChangedAt = changeDetail.String, changedAt.String
		items = append(items, item)
	}
	if err := rows.Err(); err != nil {
		return nil, apperror.Wrap(err, "遍历采购明细失败")
	}
	return items, nil
}

func (s *Store) PurchaseItem(ctx context.Context, id uint64) (response.PurchaseItem, error) {
	if err := s.ready(); err != nil {
		return response.PurchaseItem{}, err
	}
	var item response.PurchaseItem
	var changeDetail, changedAt sql.NullString
	err := s.DB.QueryRowContext(ctx, `SELECT id,DATE_FORMAT(purchase_date,'%Y-%m-%d'),supplier_name,category,product_name,CAST(quantity AS CHAR),unit,CAST(unit_price AS CHAR),CAST(total_price AS CHAR),remark,last_change_detail,DATE_FORMAT(last_changed_at,'%Y-%m-%d %H:%i:%s'),last_change_detail IS NOT NULL AND last_change_detail<>'' FROM purchase_items WHERE id=? AND active=1`, id).Scan(&item.ID, &item.Date, &item.SupplierName, &item.Category, &item.ProductName, &item.Quantity, &item.Unit, &item.UnitPrice, &item.TotalPrice, &item.Remark, &changeDetail, &changedAt, &item.Modified)
	if err == sql.ErrNoRows {
		return response.PurchaseItem{}, ErrNotFound
	}
	if err != nil {
		return response.PurchaseItem{}, apperror.Wrap(err, "读取采购明细失败")
	}
	item.ChangeDetail, item.ChangedAt = changeDetail.String, changedAt.String
	return item, nil
}

func (s *Store) PurchaseCategoryTotals(ctx context.Context, start, end string) ([]response.PurchaseCategoryTotal, error) {
	if err := s.ready(); err != nil {
		return nil, err
	}
	rows, err := s.DB.QueryContext(ctx, `SELECT category,CAST(COALESCE(SUM(total_price),0) AS CHAR) FROM purchase_items WHERE active=1 AND purchase_date>=? AND purchase_date<DATE_ADD(?,INTERVAL 1 DAY) GROUP BY category ORDER BY category`, start, end)
	if err != nil {
		return nil, apperror.Wrap(err, "统计采购品类金额失败")
	}
	defer rows.Close()
	result := make([]response.PurchaseCategoryTotal, 0)
	for rows.Next() {
		var item response.PurchaseCategoryTotal
		if err := rows.Scan(&item.Category, &item.TotalPrice); err != nil {
			return nil, apperror.Wrap(err, "读取采购品类金额失败")
		}
		result = append(result, item)
	}
	return result, rows.Err()
}

func (s *Store) SavePurchaseItem(ctx context.Context, id, actor uint64, v request.PurchaseInput) (uint64, error) {
	if err := s.ready(); err != nil {
		return 0, err
	}
	total, err := purchaseTotal(v.Quantity, v.UnitPrice)
	if err != nil {
		return 0, err
	}
	tx, err := s.DB.BeginTx(ctx, nil)
	if err != nil {
		return 0, apperror.Wrap(err, "开始保存采购明细事务失败")
	}
	defer tx.Rollback()
	if id == 0 {
		result, err := tx.ExecContext(ctx, `INSERT INTO purchase_items(purchase_date,supplier_name,category,product_name,quantity,unit,unit_price,total_price,remark,operator_id) VALUES(?,?,?,?,?,?,?,?,?,?)`, v.Date, v.SupplierName, v.Category, v.ProductName, v.Quantity, v.Unit, v.UnitPrice, total, v.Remark, optionalID(actor))
		if err != nil {
			return 0, apperror.Wrap(err, "新增采购明细失败")
		}
		id64, err := result.LastInsertId()
		if err != nil {
			return 0, apperror.Wrap(err, "读取采购明细ID失败")
		}
		id = uint64(id64)
	} else {
		var old purchaseSnapshot
		var oldTotal string
		err := tx.QueryRowContext(ctx, `SELECT DATE_FORMAT(purchase_date,'%Y-%m-%d'),supplier_name,category,product_name,CAST(quantity AS CHAR),unit,CAST(unit_price AS CHAR),CAST(total_price AS CHAR),remark FROM purchase_items WHERE id=? AND active=1 FOR UPDATE`, id).Scan(&old.Date, &old.SupplierName, &old.Category, &old.ProductName, &old.Quantity, &old.Unit, &old.UnitPrice, &oldTotal, &old.Remark)
		if err == sql.ErrNoRows {
			return 0, ErrNotFound
		}
		if err != nil {
			return 0, apperror.Wrap(err, "读取待修改采购明细失败")
		}
		old.TotalPrice = oldTotal
		changeDetail := purchaseChangeDetail(old, v, total)
		var result sql.Result
		if changeDetail != "" {
			result, err = tx.ExecContext(ctx, `UPDATE purchase_items SET purchase_date=?,supplier_name=?,category=?,product_name=?,quantity=?,unit=?,unit_price=?,total_price=?,remark=?,operator_id=?,last_change_detail=?,last_changed_at=NOW() WHERE id=? AND active=1`, v.Date, v.SupplierName, v.Category, v.ProductName, v.Quantity, v.Unit, v.UnitPrice, total, v.Remark, optionalID(actor), changeDetail, id)
		} else {
			result, err = tx.ExecContext(ctx, `UPDATE purchase_items SET purchase_date=?,supplier_name=?,category=?,product_name=?,quantity=?,unit=?,unit_price=?,total_price=?,remark=?,operator_id=? WHERE id=? AND active=1`, v.Date, v.SupplierName, v.Category, v.ProductName, v.Quantity, v.Unit, v.UnitPrice, total, v.Remark, optionalID(actor), id)
		}
		if err != nil {
			return 0, apperror.Wrap(err, "修改采购明细失败")
		}
		if affected, _ := result.RowsAffected(); affected == 0 {
			return 0, ErrNotFound
		}
	}
	if err := miniAudit(ctx, tx, actor, "purchase", id, v.ProductName); err != nil {
		return 0, apperror.Wrap(err, "记录采购明细操作日志失败")
	}
	if err := tx.Commit(); err != nil {
		return 0, apperror.Wrap(err, "提交采购明细失败")
	}
	return id, nil
}

func (s *Store) DeletePurchaseItem(ctx context.Context, id, actor uint64) error {
	if err := s.ready(); err != nil {
		return err
	}
	tx, err := s.DB.BeginTx(ctx, nil)
	if err != nil {
		return err
	}
	defer tx.Rollback()
	var name string
	if err := tx.QueryRowContext(ctx, `SELECT product_name FROM purchase_items WHERE id=? AND active=1 FOR UPDATE`, id).Scan(&name); err != nil {
		if err == sql.ErrNoRows {
			return ErrNotFound
		}
		return err
	}
	if _, err := tx.ExecContext(ctx, `UPDATE purchase_items SET active=0,deleted_at=NOW(),deleted_by=? WHERE id=? AND active=1`, optionalID(actor), id); err != nil {
		return err
	}
	if err := miniAudit(ctx, tx, actor, "purchase", id, "删除采购明细："+name); err != nil {
		return err
	}
	return tx.Commit()
}

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

func menuPageNumber(value string) int {
	n, _ := strconv.Atoi(value)
	if n < 1 {
		return 1
	}
	return n
}

func menuPageSize(value string) int {
	n, _ := strconv.Atoi(value)
	if n != 20 && n != 50 && n != 100 {
		return 20
	}
	return n
}

// MenuItems 查询菜单，默认按价格降序；筛选和分页全部在数据库完成。
func (s *Store) MenuItems(ctx context.Context, page, pageSize int, keyword, priceCategory, itemCategory, minPrice, maxPrice, sort string) (response.MenuPage, error) {
	if err := s.ready(); err != nil {
		return response.MenuPage{}, err
	}
	page = menuPageNumber(strconv.Itoa(page))
	pageSize = menuPageSize(strconv.Itoa(pageSize))
	if sort != "price_asc" {
		sort = "price_desc"
	}
	where := []string{"active=1"}
	args := make([]any, 0, 8)
	if value := strings.TrimSpace(keyword); value != "" {
		where = append(where, "(name LIKE ? OR window_name LIKE ? OR source_no LIKE ?)")
		value = "%" + value + "%"
		args = append(args, value, value, value)
	}
	if value := strings.TrimSpace(priceCategory); value != "" {
		where = append(where, "price_category=?")
		args = append(args, value)
	}
	if value := strings.TrimSpace(itemCategory); value != "" {
		where = append(where, "item_category=?")
		args = append(args, value)
	}
	if value := strings.TrimSpace(minPrice); value != "" {
		if n, err := strconv.ParseFloat(value, 64); err == nil {
			where = append(where, "price>=?")
			args = append(args, n)
		}
	}
	if value := strings.TrimSpace(maxPrice); value != "" {
		if n, err := strconv.ParseFloat(value, 64); err == nil {
			where = append(where, "price<=?")
			args = append(args, n)
		}
	}
	condition := strings.Join(where, " AND ")
	var total int
	if err := s.DB.QueryRowContext(ctx, "SELECT COUNT(*) FROM menu_items WHERE "+condition, args...).Scan(&total); err != nil {
		return response.MenuPage{}, apperror.Wrap(err, "统计菜单数量失败")
	}
	if total == 0 {
		page = 1
	}
	totalPages := int(math.Ceil(float64(total) / float64(pageSize)))
	if totalPages > 0 && page > totalPages {
		page = totalPages
	}
	offset := (page - 1) * pageSize
	orderBy := "price DESC, name ASC, id ASC"
	if sort == "price_asc" {
		orderBy = "price ASC, name ASC, id ASC"
	}
	query := `SELECT id,source_sheet,source_row,source_no,restaurant,window_name,name,specification,price,price_text,sales_period,price_range,price_category,item_category,remark FROM menu_items WHERE ` + condition + ` ORDER BY ` + orderBy + ` LIMIT ? OFFSET ?`
	listArgs := append(append([]any{}, args...), pageSize, offset)
	rows, err := s.DB.QueryContext(ctx, query, listArgs...)
	if err != nil {
		return response.MenuPage{}, apperror.Wrap(err, "查询菜单列表失败")
	}
	defer rows.Close()
	items := make([]response.MenuItem, 0, pageSize)
	for rows.Next() {
		var item response.MenuItem
		var price sql.NullFloat64
		if err := rows.Scan(&item.ID, &item.SourceSheet, &item.SourceRow, &item.SourceNo, &item.Restaurant, &item.WindowName, &item.Name, &item.Specification, &price, &item.PriceText, &item.SalesPeriod, &item.PriceRange, &item.PriceCategory, &item.ItemCategory, &item.Remark); err != nil {
			return response.MenuPage{}, apperror.Wrap(err, "读取菜单列表失败")
		}
		item.Price = Decimal(price.Float64)
		item.Seq = offset + len(items) + 1
		items = append(items, item)
	}
	if err := rows.Err(); err != nil {
		return response.MenuPage{}, apperror.Wrap(err, "遍历菜单列表失败")
	}
	priceCategories, itemCategories, err := s.menuCategories(ctx)
	if err != nil {
		return response.MenuPage{}, err
	}
	priceStats, err := s.menuPriceStats(ctx, condition, args)
	if err != nil {
		return response.MenuPage{}, err
	}
	return response.MenuPage{Items: items, Page: page, PageSize: pageSize, Total: total, TotalPages: totalPages, Sort: sort, PriceStats: priceStats, PriceCategories: priceCategories, ItemCategories: itemCategories}, nil
}

func (s *Store) menuPriceStats(ctx context.Context, condition string, args []any) ([]response.MenuPriceStat, error) {
	rows, err := s.DB.QueryContext(ctx, "SELECT price,price_text,COUNT(*) FROM menu_items WHERE "+condition+" GROUP BY price,price_text ORDER BY price DESC,price_text", args...)
	if err != nil {
		return nil, apperror.Wrap(err, "统计菜单价格数量失败")
	}
	defer rows.Close()
	result := make([]response.MenuPriceStat, 0)
	for rows.Next() {
		var price sql.NullFloat64
		var label string
		var count int
		if err := rows.Scan(&price, &label, &count); err != nil {
			return nil, apperror.Wrap(err, "读取菜单价格统计失败")
		}
		result = append(result, response.MenuPriceStat{Price: Decimal(price.Float64), PriceText: label, Count: count})
	}
	return result, rows.Err()
}

func (s *Store) menuCategories(ctx context.Context) ([]string, []string, error) {
	read := func(column string) ([]string, error) {
		rows, err := s.DB.QueryContext(ctx, "SELECT DISTINCT "+column+" FROM menu_items WHERE active=1 AND "+column+"<>'' ORDER BY "+column)
		if err != nil {
			return nil, apperror.Wrap(err, "查询菜单筛选项失败")
		}
		defer rows.Close()
		items := make([]string, 0)
		for rows.Next() {
			var value string
			if err := rows.Scan(&value); err != nil {
				return nil, err
			}
			items = append(items, value)
		}
		return items, rows.Err()
	}
	prices, err := read("price_category")
	if err != nil {
		return nil, nil, err
	}
	categories, err := read("item_category")
	return prices, categories, err
}

func (s *Store) SaveMenuItem(ctx context.Context, id, actor uint64, v request.MenuItemInput) (uint64, error) {
	if err := s.ready(); err != nil {
		return 0, err
	}
	values := []any{v.Restaurant, v.WindowName, v.Name, v.Specification, v.Price, v.PriceText, v.SalesPeriod, v.PriceRange, v.PriceCategory, v.ItemCategory, v.Remark}
	if id == 0 {
		result, err := s.DB.ExecContext(ctx, `INSERT INTO menu_items(restaurant,window_name,name,specification,price,price_text,sales_period,price_range,price_category,item_category,remark,source_sheet,source_row,source_no) VALUES(?,?,?,?,?,?,?,?,?,?,?,'手工录入',0,'')`, values...)
		if err != nil {
			return 0, apperror.Wrap(err, "新增菜单失败")
		}
		last, err := result.LastInsertId()
		if err != nil {
			return 0, err
		}
		id = uint64(last)
	} else {
		values = append(values, id)
		result, err := s.DB.ExecContext(ctx, `UPDATE menu_items SET restaurant=?,window_name=?,name=?,specification=?,price=?,price_text=?,sales_period=?,price_range=?,price_category=?,item_category=?,remark=? WHERE id=? AND active=1`, values...)
		if err != nil {
			return 0, apperror.Wrap(err, "修改菜单失败")
		}
		affected, _ := result.RowsAffected()
		if affected == 0 {
			return 0, ErrNotFound
		}
	}
	_, err := s.DB.ExecContext(ctx, `INSERT INTO audit_logs(admin_user_id,module,action,target_type,target_id,target_text,detail) VALUES(?,?,?,?,?,?,?)`, actor, "menu", "保存菜单", "menu_items", id, v.Name, fmt.Sprintf("窗口：%s", v.WindowName))
	return id, err
}

func (s *Store) DeleteMenuItem(ctx context.Context, id, actor uint64) error {
	if err := s.ready(); err != nil {
		return err
	}
	result, err := s.DB.ExecContext(ctx, `UPDATE menu_items SET active=0 WHERE id=? AND active=1`, id)
	if err != nil {
		return apperror.Wrap(err, "删除菜单失败")
	}
	affected, _ := result.RowsAffected()
	if affected == 0 {
		return ErrNotFound
	}
	_, err = s.DB.ExecContext(ctx, `INSERT INTO audit_logs(admin_user_id,module,action,target_type,target_id,target_text,detail) VALUES(?,?,?,?,?,?,?)`, actor, "menu", "删除菜单", "menu_items", id, "菜单记录", "软删除")
	return err
}

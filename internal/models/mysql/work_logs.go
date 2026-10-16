package mysql

import (
	"context"
	"database/sql"
	"errors"
	"strings"

	"friends-records/api/request"
	"friends-records/api/response"
	"friends-records/internal/apperror"
)

func (s *Store) WorkLogs(ctx context.Context, month, search string) ([]response.WorkLog, error) {
	if err := s.ready(); err != nil {
		return nil, err
	}
	month = NormalizeMonth(month)
	search = stringsTrim(search)
	rows, err := s.DB.QueryContext(ctx, `SELECT w.id,w.employee_id,e.employee_no,e.name,DATE_FORMAT(w.work_date,'%Y-%m-%d'),w.hours,e.pay_basis,COALESCE(e.current_salary,0),w.remark FROM employee_work_logs w JOIN employees e ON e.id=w.employee_id WHERE w.active=1 AND e.active=1 AND e.employment_type='part_time' AND e.pay_basis IN ('hourly','daily','half_daily') AND w.work_date>=? AND w.work_date<DATE_ADD(?,INTERVAL 1 MONTH) AND (?='' OR LOCATE(?,e.name)>0 OR LOCATE(?,e.employee_no)>0) ORDER BY w.work_date DESC,e.employee_no`, month+"-01", month+"-01", search, search, search)
	if err != nil {
		return nil, apperror.Wrap(err, "查询小时工工时失败")
	}
	defer rows.Close()
	items := make([]response.WorkLog, 0)
	for rows.Next() {
		var item response.WorkLog
		var hours, rate float64
		if err := rows.Scan(&item.ID, &item.EmployeeID, &item.EmployeeNo, &item.Name, &item.WorkDate, &hours, &item.PayBasis, &rate, &item.Remark); err != nil {
			return nil, apperror.Wrap(err, "读取小时工工时失败")
		}
		item.Hours, item.HoursValue = Decimal(hours), Decimal(hours)
		item.PayRate = Money(rate)
		var amount float64
		item.PayBasisLabel, amount = payBasisAmount(item.PayBasis, rate, hours)
		item.Amount, item.AmountValue = Money(amount), Decimal(amount)
		items = append(items, item)
	}
	if err := rows.Err(); err != nil {
		return nil, apperror.Wrap(err, "遍历小时工工时失败")
	}
	return items, nil
}

func (s *Store) SaveWorkLog(ctx context.Context, id, actor uint64, v request.WorkLogInput) (uint64, error) {
	if err := s.ready(); err != nil {
		return 0, err
	}
	if err := v.Validate(); err != nil {
		return 0, apperror.New(err.Error())
	}
	tx, err := s.DB.BeginTx(ctx, nil)
	if err != nil {
		return 0, apperror.Wrap(err, "开始保存工时事务失败")
	}
	defer tx.Rollback()
	var payBasis, employmentType string
	if err := tx.QueryRowContext(ctx, `SELECT pay_basis,employment_type FROM employees WHERE id=? AND active=1 FOR SHARE`, v.EmployeeID).Scan(&payBasis, &employmentType); err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return 0, ErrNotFound
		}
		return 0, apperror.Wrap(err, "读取小时工资料失败")
	}
	if employmentType != "part_time" || (payBasis != "hourly" && payBasis != "daily" && payBasis != "half_daily") {
		return 0, apperror.New("只有按小时、按天或按半天计薪的临时工可以录入每日工时")
	}
	if id == 0 {
		// 同一员工同一天保持一条记录；已删除的旧记录会被直接恢复，避免唯一键阻止重新补录。
		result, err := tx.ExecContext(ctx, `INSERT INTO employee_work_logs(employee_id,work_date,hours,remark) VALUES(?,?,?,?) ON DUPLICATE KEY UPDATE id=LAST_INSERT_ID(id),hours=VALUES(hours),remark=VALUES(remark),active=1,deleted_at=NULL,deleted_by=NULL`, v.EmployeeID, v.WorkDate, v.Hours, v.Remark)
		if err != nil {
			return 0, apperror.Wrap(err, "新增每日工时失败")
		}
		last, err := result.LastInsertId()
		if err != nil {
			return 0, apperror.Wrap(err, "读取每日工时编号失败")
		}
		id = uint64(last)
	} else {
		result, err := tx.ExecContext(ctx, `UPDATE employee_work_logs SET employee_id=?,work_date=?,hours=?,remark=? WHERE id=? AND active=1`, v.EmployeeID, v.WorkDate, v.Hours, v.Remark, id)
		if err != nil {
			return 0, apperror.Wrap(err, "修改每日工时失败")
		}
		count, _ := result.RowsAffected()
		if count == 0 {
			return 0, ErrNotFound
		}
	}
	if err := miniAudit(ctx, tx, actor, "work_logs", id, v.WorkDate+" "+v.Hours+"小时"); err != nil {
		return 0, err
	}
	if err := tx.Commit(); err != nil {
		return 0, apperror.Wrap(err, "提交每日工时失败")
	}
	return id, nil
}

// payBasisAmount 将统一保存的小时数换算成当天金额，全天=8小时，半天=4小时。
func payBasisAmount(payBasis string, rate, hours float64) (string, float64) {
	switch payBasis {
	case "daily":
		return "按天", rate * hours / 8
	case "half_daily":
		return "按半天", rate * hours / 4
	default:
		return "按小时", rate * hours
	}
}

func (s *Store) DeleteWorkLog(ctx context.Context, id, actor uint64) error {
	_, err := s.DeleteWorkLogs(ctx, []uint64{id}, actor)
	return err
}

// DeleteWorkLogs 软删除多条每日工时，确保批量操作要么全部成功要么全部回滚。
func (s *Store) DeleteWorkLogs(ctx context.Context, ids []uint64, actor uint64) (int, error) {
	if err := s.ready(); err != nil {
		return 0, err
	}
	seen := make(map[uint64]struct{}, len(ids))
	validIDs := make([]uint64, 0, len(ids))
	for _, id := range ids {
		if id == 0 {
			continue
		}
		if _, exists := seen[id]; !exists {
			seen[id] = struct{}{}
			validIDs = append(validIDs, id)
		}
	}
	if len(validIDs) == 0 {
		return 0, MiniInputError{"请选择要删除的工时记录"}
	}
	tx, err := s.DB.BeginTx(ctx, nil)
	if err != nil {
		return 0, apperror.Wrap(err, "开始删除工时事务失败")
	}
	defer tx.Rollback()
	for _, id := range validIDs {
		var date string
		if err := tx.QueryRowContext(ctx, `SELECT DATE_FORMAT(work_date,'%Y-%m-%d') FROM employee_work_logs WHERE id=? AND active=1 FOR UPDATE`, id).Scan(&date); err != nil {
			if errors.Is(err, sql.ErrNoRows) {
				return 0, ErrNotFound
			}
			return 0, apperror.Wrap(err, "读取每日工时失败")
		}
		if _, err := tx.ExecContext(ctx, `UPDATE employee_work_logs SET active=0,deleted_at=NOW(),deleted_by=? WHERE id=? AND active=1`, actor, id); err != nil {
			return 0, apperror.Wrap(err, "删除每日工时失败")
		}
		if err := miniAudit(ctx, tx, actor, "work_logs", id, "删除工时："+date); err != nil {
			return 0, err
		}
	}
	if err := tx.Commit(); err != nil {
		return 0, apperror.Wrap(err, "提交删除工时事务失败")
	}
	return len(validIDs), nil
}

func stringsTrim(value string) string {
	return strings.TrimSpace(value)
}

package mysql

import (
	"context"
	"fmt"
	"math"
	"time"

	"friends-records/api/response"
	"friends-records/internal/apperror"
)

func (s *Store) Ledger(ctx context.Context, month string) ([]response.LedgerRecord, error) {
	if err := s.ready(); err != nil {
		return nil, err
	}
	month = NormalizeMonth(month)
	rows, err := s.DB.QueryContext(ctx, `SELECT l.id,DATE_FORMAT(l.occurred_on,'%Y-%m-%d'),COALESCE(d.name,''),l.summary,l.counterparty,DATE_FORMAT(l.occurred_on,'%c月'),a.name,COALESCE(c.name,''),l.direction,l.amount,a.opening_balance+COALESCE((SELECT SUM(IF(x.direction='income',x.amount,-x.amount)) FROM ledger_entries x WHERE x.account_id=l.account_id AND x.active=1 AND (x.occurred_on<l.occurred_on OR (x.occurred_on=l.occurred_on AND x.id<=l.id))),0),l.voucher_no,COALESCE(pb.id,0) FROM ledger_entries l JOIN ledger_accounts a ON a.id=l.account_id LEFT JOIN departments d ON d.id=l.department_id LEFT JOIN ledger_categories c ON c.id=l.category_id LEFT JOIN payroll_batches pb ON pb.ledger_entry_id=l.id WHERE l.active=1 AND l.occurred_on>=? AND l.occurred_on<DATE_ADD(?,INTERVAL 1 MONTH) ORDER BY l.occurred_on,l.id`, month+"-01", month+"-01")
	if err != nil {
		return nil, apperror.Wrap(err, "查询台账流水失败")
	}
	defer rows.Close()
	result := make([]response.LedgerRecord, 0)
	for rows.Next() {
		var v response.LedgerRecord
		var direction string
		var amount, balance float64
		if err := rows.Scan(&v.ID, &v.Date, &v.Department, &v.Summary, &v.Counterparty, &v.Month, &v.Account, &v.Category, &direction, &amount, &balance, &v.VoucherNo, &v.PayrollBatchID); err != nil {
			return nil, apperror.Wrap(err, "读取台账流水失败")
		}
		v.No = len(result) + 1
		if direction == "income" {
			v.Income, v.Expense = Money(amount), "—"
		} else {
			v.Income, v.Expense = "—", Money(amount)
		}
		v.Balance = Money(balance)
		result = append(result, v)
	}
	if err := rows.Err(); err != nil {
		return nil, apperror.Wrap(err, "遍历台账流水失败")
	}
	return result, nil
}

func (s *Store) LedgerSummary(ctx context.Context, month string) (response.LedgerSummary, error) {
	if err := s.ready(); err != nil {
		return response.LedgerSummary{}, err
	}
	month = NormalizeMonth(month)
	year := month[:4]
	var v response.LedgerSummary
	var mi, me, balance, yi, ye, maxi, maxe float64
	err := s.DB.QueryRowContext(ctx, `SELECT COALESCE(SUM(CASE WHEN l.occurred_on>=? AND l.occurred_on<DATE_ADD(?,INTERVAL 1 MONTH) AND l.direction='income' THEN l.amount ELSE 0 END),0),COALESCE(SUM(CASE WHEN l.occurred_on>=? AND l.occurred_on<DATE_ADD(?,INTERVAL 1 MONTH) AND l.direction='expense' THEN l.amount ELSE 0 END),0),COALESCE((SELECT SUM(opening_balance) FROM ledger_accounts),0)+COALESCE(SUM(CASE WHEN l.occurred_on<DATE_ADD(?,INTERVAL 1 MONTH) THEN IF(l.direction='income',l.amount,-l.amount) ELSE 0 END),0),COALESCE(SUM(CASE WHEN YEAR(l.occurred_on)=? AND l.direction='income' THEN l.amount ELSE 0 END),0),COALESCE(SUM(CASE WHEN YEAR(l.occurred_on)=? AND l.direction='expense' THEN l.amount ELSE 0 END),0),COALESCE(MAX(CASE WHEN YEAR(l.occurred_on)=? AND l.direction='income' THEN l.amount END),0),COALESCE(MAX(CASE WHEN YEAR(l.occurred_on)=? AND l.direction='expense' THEN l.amount END),0),COALESCE(SUM(YEAR(l.occurred_on)=?),0) FROM ledger_entries l WHERE l.active=1`, month+"-01", month+"-01", month+"-01", month+"-01", month+"-01", year, year, year, year, year).Scan(&mi, &me, &balance, &yi, &ye, &maxi, &maxe, &v.EntryCount)
	if err != nil {
		return response.LedgerSummary{}, apperror.Wrap(err, "统计台账数据失败")
	}
	v.Month = month
	v.MonthIncome, v.MonthExpense, v.MonthNet, v.CurrentBalance = Money(mi), Money(me), Money(mi-me), Money(balance)
	v.YearIncome, v.YearExpense, v.YearNet, v.MaxIncome, v.MaxExpense = Money(yi), Money(ye), Money(yi-ye), Money(maxi), Money(maxe)
	return v, nil
}

func (s *Store) LedgerTrend(ctx context.Context, year int) ([]response.TrendMonth, error) {
	if err := s.ready(); err != nil {
		return nil, err
	}
	rows, err := s.DB.QueryContext(ctx, `SELECT MONTH(occurred_on),COALESCE(SUM(IF(direction='income',amount,0)),0),COALESCE(SUM(IF(direction='expense',amount,0)),0) FROM ledger_entries WHERE YEAR(occurred_on)=? AND active=1 GROUP BY MONTH(occurred_on)`, year)
	if err != nil {
		return nil, apperror.Wrap(err, "查询年度收支趋势失败")
	}
	defer rows.Close()
	type pair struct{ income, expense float64 }
	values := make(map[int]pair)
	max := 0.0
	for rows.Next() {
		var month int
		var v pair
		if err := rows.Scan(&month, &v.income, &v.expense); err != nil {
			return nil, apperror.Wrap(err, "读取年度收支趋势失败")
		}
		values[month] = v
		if v.income > max {
			max = v.income
		}
		if v.expense > max {
			max = v.expense
		}
	}
	if err := rows.Err(); err != nil {
		return nil, apperror.Wrap(err, "遍历年度收支趋势失败")
	}
	result := make([]response.TrendMonth, 0, 12)
	for month := 1; month <= 12; month++ {
		v := values[month]
		ip, ep := 0, 0
		if max > 0 {
			ip = int(math.Round(v.income / max * 100))
			ep = int(math.Round(v.expense / max * 100))
		}
		result = append(result, response.TrendMonth{Month: fmt.Sprintf("%d月", month), Income: Money(v.income), Expense: Money(v.expense), IncomePercent: ip, ExpensePercent: ep})
	}
	return result, nil
}

func LedgerYear(month string) int {
	value, _ := time.Parse("2006-01", NormalizeMonth(month))
	return value.Year()
}

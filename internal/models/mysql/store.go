package mysql

import (
	"database/sql"
	"fmt"
	"strings"

	"friends-records/internal/apperror"
)

var ErrNotConfigured = apperror.New("数据库尚未配置")
var ErrNotFound = apperror.New("数据不存在")

type Store struct {
	DB              *sql.DB
	MonthlyRestDays int
}

func New(db *sql.DB) *Store { return &Store{DB: db, MonthlyRestDays: 3} }

func (s *Store) ready() error {
	if s.DB == nil {
		return ErrNotConfigured
	}
	return nil
}

func Money(value float64) string   { return fmt.Sprintf("¥%.2f", value) }
func Decimal(value float64) string { return fmt.Sprintf("%.2f", value) }

func Mask(value string, left, right int) string {
	runes := []rune(value)
	if value == "" || len(runes) <= left+right {
		return value
	}
	return string(runes[:left]) + strings.Repeat("*", len(runes)-left-right) + string(runes[len(runes)-right:])
}

func Status(value string) (string, string) {
	switch value {
	case "active":
		return "在职", "active"
	case "left":
		return "已离职", "inactive"
	default:
		return "试用期", "pending"
	}
}

func PayrollStatus(value string) (string, string) {
	switch value {
	case "paid":
		return "已发放", "active"
	case "confirmed":
		return "已确认", "warning"
	default:
		return "待确认", "pending"
	}
}

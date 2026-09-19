package service

import (
	"context"

	"friends-records/api/request"
	"friends-records/internal/models/mysql"
)

type Payroll struct{ Store *mysql.Store }

func (s Payroll) Update(ctx context.Context, id uint64, input request.PayrollUpdate) (float64, error) {
	return s.Store.UpdatePayroll(ctx, id, input)
}
func (s Payroll) Generate(ctx context.Context, month string) (uint64, error) {
	return s.Store.GeneratePayroll(ctx, month)
}
func (s Payroll) GenerateEmployee(ctx context.Context, month string, employeeID uint64) (uint64, error) {
	return s.Store.GeneratePayrollForEmployee(ctx, month, employeeID)
}
func (s Payroll) Pay(ctx context.Context, input request.PayrollPay) (uint64, error) {
	return s.Store.PayPayroll(ctx, input.BatchID, input.AccountID, input.OccurredOn)
}
func (s Payroll) PayEmployee(ctx context.Context, input request.PayrollPay) (uint64, error) {
	return s.Store.PayPayrollItem(ctx, input.ItemID, input.AccountID, input.OccurredOn)
}
func (s Payroll) Confirm(ctx context.Context, batchID uint64) error {
	return s.Store.ConfirmPayroll(ctx, batchID)
}
func (s Payroll) ConfirmEmployee(ctx context.Context, itemID uint64) error {
	return s.Store.ConfirmPayrollItem(ctx, itemID)
}

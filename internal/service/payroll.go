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
func (s Payroll) Pay(ctx context.Context, input request.PayrollPay) (uint64, error) {
	return s.Store.PayPayroll(ctx, input.BatchID, input.AccountID, input.OccurredOn)
}
func (s Payroll) Confirm(ctx context.Context, batchID uint64) error {
	return s.Store.ConfirmPayroll(ctx, batchID)
}

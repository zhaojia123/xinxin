package service

import (
	"context"

	"friends-records/api/request"
	"friends-records/api/response"
	"friends-records/internal/models/mysql"
)

type Payroll struct{ Store *mysql.Store }

func (s Payroll) Update(ctx context.Context, id uint64, input request.PayrollUpdate) (float64, error) {
	return s.Store.UpdatePayroll(ctx, id, input)
}
func (s Payroll) Delete(ctx context.Context, ids []uint64, actor uint64) (int, error) {
	return s.Store.DeletePayrollItems(ctx, ids, actor)
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
	return s.Store.PayPayrollItem(ctx, input.ItemID, input.AccountID, input.OccurredOn, input.PaymentRemark)
}
func (s Payroll) PayEmployees(ctx context.Context, input request.PayrollPay) (uint64, error) {
	return s.Store.PayPayrollItems(ctx, input.ItemIDs, input.AccountID, input.OccurredOn, input.PaymentRemarks)
}
func (s Payroll) PayTemporary(ctx context.Context, input request.TemporaryPayrollInput) (uint64, error) {
	return s.Store.PayTemporaryPayroll(ctx, input)
}
func (s Payroll) Confirm(ctx context.Context, batchID uint64) error {
	return s.Store.ConfirmPayroll(ctx, batchID)
}
func (s Payroll) ConfirmEmployee(ctx context.Context, itemID uint64) error {
	return s.Store.ConfirmPayrollItem(ctx, itemID)
}
func (s Payroll) SettleType(ctx context.Context, actor uint64, input request.PayrollSettlementInput) (uint64, error) {
	return s.Store.SettlePayrollType(ctx, actor, input)
}
func (s Payroll) RevokeSettlement(ctx context.Context, actor uint64, input request.PayrollSettlementRevokeInput) error {
	return s.Store.RevokePayrollSettlement(ctx, actor, input)
}
func (s Payroll) SpecialList(ctx context.Context, month string) ([]response.SpecialPayrollRecord, response.SpecialPayrollSummary, error) {
	return s.Store.SpecialPayrolls(ctx, month)
}
func (s Payroll) SaveSpecial(ctx context.Context, id, actor uint64, input request.SpecialPayrollInput) (uint64, error) {
	return s.Store.SaveSpecialPayroll(ctx, id, actor, input)
}
func (s Payroll) DeleteSpecial(ctx context.Context, id, actor uint64) error {
	return s.Store.DeleteSpecialPayroll(ctx, id, actor)
}
func (s Payroll) SettleSpecial(ctx context.Context, actor uint64, input request.SpecialPayrollSettlementInput) (uint64, error) {
	return s.Store.SettleSpecialPayroll(ctx, actor, input)
}

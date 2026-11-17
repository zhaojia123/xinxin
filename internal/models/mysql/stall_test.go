package mysql

import (
	"testing"

	"friends-records/api/response"
)

func TestCalculateStallMonthlyRentOverridesLegacyManualCommission(t *testing.T) {
	item := response.Stall{
		AllocationMode:         "custom",
		MonthlyRent:            "20000.00",
		SchoolCardAmount:       "162155.19",
		HuilaimiAmount:         "0.00",
		CostAmount:             "0.00",
		ManualOwnCommission:    "0.00",
		ManualOwnCommissionSet: true,
	}

	calculateStall(&item, 0.04)

	if item.OwnCommission != "20000.00" {
		t.Fatalf("月租规则不应被历史手动金额覆盖，实际我方抽成：%s", item.OwnCommission)
	}
	if item.CanManualOwnCommission {
		t.Fatal("已配置月租时不应允许手动填写我方抽成")
	}
	if item.NetTurnover != "162155.19" {
		t.Fatalf("档口列表不应扣康莱德后的金额错误：%s", item.NetTurnover)
	}
}

func TestCalculateStallAllowsManualCommissionWithoutRule(t *testing.T) {
	item := response.Stall{
		AllocationMode:         "custom",
		SchoolCardAmount:       "100.00",
		ManualOwnCommission:    "12.50",
		ManualOwnCommissionSet: true,
	}

	calculateStall(&item, 0)

	if item.OwnCommission != "12.50" || item.JiaCommission != "11.25" || item.YiCommission != "1.25" || !item.CanManualOwnCommission {
		t.Fatalf("未配置规则时应使用手动抽成并按90/10拆分，抽成=%s，甲=%s，乙=%s，可编辑=%t", item.OwnCommission, item.JiaCommission, item.YiCommission, item.CanManualOwnCommission)
	}
}

func TestCalculateStallDefaultsOtherStallToFullNetAmount(t *testing.T) {
	item := response.Stall{
		AllocationMode:   "custom",
		SchoolCardAmount: "100.00",
		HuilaimiAmount:   "20.00",
	}

	calculateStall(&item, defaultConradRate)

	// 100 + 20 = 120；未设抽点规则时全额按甲乙90/10分配。
	if item.OwnCommission != "120.00" || item.JiaCommission != "108.00" || item.YiCommission != "12.00" {
		t.Fatalf("未设规则时应按可计算额全额分配，实际总额=%s，甲=%s，乙=%s", item.OwnCommission, item.JiaCommission, item.YiCommission)
	}
}

func TestCalculateStallUsesAfterConradAmountForOtherStallRate(t *testing.T) {
	item := response.Stall{
		AllocationMode:   "custom",
		SchoolCardAmount: "100.00",
		HuilaimiAmount:   "100.00",
		TotalAmountRate:  "10.00",
	}

	calculateStall(&item, defaultConradRate)

	// 档口列表不扣康莱德，(100 + 100) × 10% = 20；康莱德在月度汇总中统一扣除。
	if item.OwnCommission != "20.00" {
		t.Fatalf("非自营档口抽成不应在明细中扣康莱德，实际：%s", item.OwnCommission)
	}
}

func TestCalculateSelfOperatedStallDeductsCostBeforeFixedSplit(t *testing.T) {
	item := response.Stall{
		AllocationMode:   "profit_share",
		SchoolCardAmount: "100.00",
		HuilaimiAmount:   "100.00",
		CostAmount:       "97.00",
	}

	calculateStall(&item, defaultConradRate)

	// (100 + 100) - 97 = 103，再按固定90/10拆分；康莱德在月度汇总中统一扣除。
	if item.OwnCommission != "103.00" || item.JiaCommission != "92.70" || item.YiCommission != "10.30" || item.Remaining != "103.00" {
		t.Fatalf("自营档口应先扣成本再按90/10分配，并将可分成利润计入交付金额，总额=%s，甲=%s，乙=%s，交付=%s", item.OwnCommission, item.JiaCommission, item.YiCommission, item.Remaining)
	}
}

func TestCalculateSelfOperatedStallDoesNotDeductRiceOilTwice(t *testing.T) {
	item := response.Stall{
		AllocationMode:    "profit_share",
		SchoolCardAmount:  "147485.11",
		CostAmount:        "115563.91",
		RiceOilCostAmount: "18785.00",
		CouponAmount:      "4480.00",
	}

	calculateStall(&item, 0)

	// 自营米油已包含在食材成本内，交付金额只扣食材再加优惠券。
	if item.Remaining != "36401.20" {
		t.Fatalf("自营档口不应重复扣米油，实际交付=%s", item.Remaining)
	}
}

func TestCalculateSelfOperatedStallKeepsLoss(t *testing.T) {
	item := response.Stall{
		AllocationMode:   "profit_share",
		SchoolCardAmount: "73665.38",
		CostAmount:       "93796.07",
	}

	calculateStall(&item, 0)

	if item.NetTurnover != "73665.38" || item.CostDeductedTurnover != "-20130.69" {
		t.Fatalf("成本前后金额应保留真实值，扣点后=%s，扣成本后=%s", item.NetTurnover, item.CostDeductedTurnover)
	}
	if item.OwnCommission != "-20130.69" || item.JiaCommission != "-18117.62" || item.YiCommission != "-2013.07" || item.Remaining != "-20130.69" {
		t.Fatalf("自营档口亏损不能归零，总额=%s，甲=%s，乙=%s，余额=%s", item.OwnCommission, item.JiaCommission, item.YiCommission, item.Remaining)
	}
}

func TestCalculateOtherStallCostDeductsRemaining(t *testing.T) {
	item := response.Stall{
		AllocationMode:   "custom",
		SchoolCardAmount: "100.00",
		HuilaimiAmount:   "100.00",
		CostAmount:       "50.00",
		TotalAmountRate:  "20.00",
	}

	calculateStall(&item, defaultConradRate)

	// 可计算额 200，按 20% 抽成 40；非自营成本必须从实际需交付档口金额中扣除。
	if item.OwnCommission != "40.00" || item.CostDeductedTurnover != "150.00" || item.Remaining != "110.00" {
		t.Fatalf("非自营成本应扣实际需交付档口金额，抽成=%s，扣成本后=%s，金额=%s", item.OwnCommission, item.CostDeductedTurnover, item.Remaining)
	}
}

func TestCalculateStallAdjustsActualDeliveryByExtraCosts(t *testing.T) {
	item := response.Stall{
		AllocationMode:    "custom",
		SchoolCardAmount:  "100.00",
		HuilaimiAmount:    "100.00",
		TotalAmountRate:   "20.00",
		AdvertisingAmount: "10.00",
		UniformAmount:     "5.00",
		CouponAmount:      "3.00",
	}

	calculateStall(&item, defaultConradRate)

	// 原交付额 160，减广告 10、减工装 5、加优惠券 3，得到 148；档口可分成利润不受影响。
	if item.Remaining != "148.00" || item.OwnCommission != "40.00" {
		t.Fatalf("实际需交付金额应计入广告、工装和优惠券调整，交付=%s，利润=%s", item.Remaining, item.OwnCommission)
	}
}

func TestCalculateDailyFreshFruitDeliveryUsesSchoolCardOnly(t *testing.T) {
	item := response.Stall{
		StallNo:          "S01",
		Name:             "每日鲜果",
		AllocationMode:   "custom",
		SchoolCardAmount: "100.00",
		HuilaimiAmount:   "20.00",
		TotalAmountRate:  "10.00",
	}

	calculateStall(&item, defaultConradRate)

	// 每日鲜果实际需交付金额只取学校卡机减档口可分成利润，汇来米不参与。
	if item.OwnCommission != "12.00" || item.Remaining != "88.00" {
		t.Fatalf("每日鲜果交付金额应为学校卡机减档口利润，利润=%s，交付=%s", item.OwnCommission, item.Remaining)
	}
}

func TestApplyMonthlyOperatingExpenseCalculatesProfitShareAfterExpenses(t *testing.T) {
	summary := response.StallSummary{
		SelfOperatedPayableAmount: "10000.00",
	}
	expense := response.StallMonthlyExpense{TotalAmount: "2500.00"}

	ApplyMonthlyOperatingExpense(&summary, expense)

	if summary.DistributableBalance != "7500.00" {
		t.Fatalf("可分配结余应先扣工资水电，实际：%s", summary.DistributableBalance)
	}
	if summary.JiaCommission != "6750.00" || summary.YiCommission != "750.00" {
		t.Fatalf("甲乙应按扣支出后的结余90/10分配，实际甲=%s，乙=%s", summary.JiaCommission, summary.YiCommission)
	}
	if summary.OwnCommission != "7500.00" {
		t.Fatalf("我方抽成应为扣除经营支出后的特殊档口利润，实际：%s", summary.OwnCommission)
	}
}

func TestApplyMonthlyOperatingExpenseAddsAdvertisingAndUniformCosts(t *testing.T) {
	summary := response.StallSummary{
		SelfOperatedPayableAmount: "10000.00",
		TotalAdvertisingAmount:    "800.00",
		TotalUniformAmount:        "200.00",
	}
	expense := response.StallMonthlyExpense{TotalAmount: "2500.00"}

	ApplyMonthlyOperatingExpense(&summary, expense)

	if summary.DistributableBalance != "8500.00" {
		t.Fatalf("盈亏应加上广告和工装费用，实际：%s", summary.DistributableBalance)
	}
	if summary.JiaCommission != "7650.00" || summary.YiCommission != "850.00" {
		t.Fatalf("加上广告和工装后应按比例分成，实际甲=%s，乙=%s", summary.JiaCommission, summary.YiCommission)
	}
}

func TestApplyMonthlyOperatingExpenseUsesPayableFormula(t *testing.T) {
	summary := response.StallSummary{
		SelfOperatedPayableAmount: "10000.00",
		OtherStallCommission:      "2000.00",
		OtherStallRiceOilCost:     "130.00",
		TotalAdvertisingAmount:    "800.00",
		TotalUniformAmount:        "200.00",
	}
	expense := response.StallMonthlyExpense{TotalAmount: "2500.00"}

	ApplyMonthlyOperatingExpense(&summary, expense)

	// 10000（自营实际交付）+ 2000（其他档口抽成）+ 130（米油）+ 800（广告）+ 200（工装）- 2500（水电工资）。
	if summary.DistributableBalance != "10630.00" || summary.CommissionBeforeOperatingExpense != "12000.00" {
		t.Fatalf("盈亏应使用实际交付口径，结余=%s，扣支出前=%s", summary.DistributableBalance, summary.CommissionBeforeOperatingExpense)
	}
}

func TestApplyMonthlyOperatingExpenseUsesSeptemberAllocationFormula(t *testing.T) {
	summary := response.StallSummary{
		SelfOperatedPayableAmount: "68000.00",
		OtherStallCommission:      "20000.00",
	}
	expense := response.StallMonthlyExpense{Month: "2026-09", TotalAmount: "8000.00"}

	ApplyMonthlyOperatingExpense(&summary, expense)

	// 自营档口已在单档计算时扣成本：68000 + 20000 - 8000 = 80000。
	if summary.DistributableBalance != "80000.00" {
		t.Fatalf("九月可分配利润应由自营净营业额加其他档口抽成后扣经营支出，实际：%s", summary.DistributableBalance)
	}
	if summary.JiaCommission != "72000.00" || summary.YiCommission != "8000.00" {
		t.Fatalf("九月甲乙应按90/10拆分，实际甲=%s，乙=%s", summary.JiaCommission, summary.YiCommission)
	}
}

func TestApplyMonthlyOperatingExpenseKeepsMonthlyLoss(t *testing.T) {
	summary := response.StallSummary{SelfOperatedPayableAmount: "100.00"}
	expense := response.StallMonthlyExpense{TotalAmount: "150.00"}

	ApplyMonthlyOperatingExpense(&summary, expense)

	if summary.DistributableBalance != "-50.00" || !summary.HasMonthlyLoss || summary.MonthlyLoss != "50.00" {
		t.Fatalf("亏损应保留负数并单独展示，结果=%s，亏损=%s，标记=%t", summary.DistributableBalance, summary.MonthlyLoss, summary.HasMonthlyLoss)
	}
	if summary.JiaCommission != "0.00" || summary.YiCommission != "0.00" {
		t.Fatalf("亏损月份不应产生甲乙抽成，甲=%s，乙=%s", summary.JiaCommission, summary.YiCommission)
	}
}

func TestApplyMonthlyOperatingExpenseDeductsConradFromMonthlyProfitAndLoss(t *testing.T) {
	summary := response.StallSummary{SelfOperatedPayableAmount: "1000.00", ConradCommission: "30.00"}
	expense := response.StallMonthlyExpense{TotalAmount: "70.00"}

	ApplyMonthlyOperatingExpense(&summary, expense)

	if summary.DistributableBalance != "900.00" || summary.JiaCommission != "810.00" || summary.YiCommission != "90.00" {
		t.Fatalf("康莱德应作为支出计入本月盈亏，结余=%s，甲=%s，乙=%s", summary.DistributableBalance, summary.JiaCommission, summary.YiCommission)
	}
}

func TestApplyMonthlyOperatingExpenseAddsBackOtherStallRiceOilCost(t *testing.T) {
	summary := response.StallSummary{
		SelfOperatedPayableAmount: "1000.00",
		OtherStallRiceOilCost:     "130.00",
		ConradCommission:          "30.00",
	}
	expense := response.StallMonthlyExpense{TotalAmount: "1150.00"}

	ApplyMonthlyOperatingExpense(&summary, expense)

	if summary.DistributableBalance != "-50.00" || !summary.HasMonthlyLoss || summary.MonthlyLoss != "50.00" {
		t.Fatalf("非自营米油应回加且康莱德应扣入本月盈亏，结余=%s，亏损=%s，标记=%t", summary.DistributableBalance, summary.MonthlyLoss, summary.HasMonthlyLoss)
	}
}

func TestApplyMonthlyOperatingExpenseUsesMonthlyLossRiceOilReconciliation(t *testing.T) {
	summary := response.StallSummary{
		SelfOperatedPayableAmount: "48335.69",
		OtherStallCommission:      "49802.60",
		OtherStallRiceOilCost:     "5355.00",
		ConradCommission:          "11883.26",
	}
	expense := response.StallMonthlyExpense{TotalAmount: "122480.06"}

	ApplyMonthlyOperatingExpense(&summary, expense)

	if summary.DistributableBalance != "-30870.03" || summary.MonthlyLoss != "30870.03" || !summary.HasMonthlyLoss {
		t.Fatalf("本月亏损应扣康莱德并回加非自营米油，结余=%s，亏损=%s，标记=%t", summary.DistributableBalance, summary.MonthlyLoss, summary.HasMonthlyLoss)
	}
}

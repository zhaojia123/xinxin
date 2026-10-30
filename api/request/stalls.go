package request

import (
	"fmt"
	"strconv"
	"strings"
	"time"
)

// StallInput 是档口基础资料；负责人直接存档口记录，不单独建人员表。
type StallInput struct {
	StallNo                 string `json:"stall_no"`
	Name                    string `json:"name"`
	ResponsibleName         string `json:"responsible_name"`
	ResponsibleMobile       string `json:"responsible_mobile"`
	MonthlyRent             string `json:"monthly_rent"`
	FixedCommissionAmount   string `json:"fixed_commission_amount"`
	DefaultGuaranteedAmount string `json:"default_guaranteed_amount"`
	AllocationMode          string `json:"allocation_mode"`
	ProfitJiaRate           string `json:"profit_jia_rate"`
	ProfitYiRate            string `json:"profit_yi_rate"`
	GuaranteedRate          string `json:"guaranteed_rate"`
	BaseAmountRate          string `json:"base_amount_rate"`
	BaseAmountFactor        string `json:"base_amount_factor"`
	TotalAmountRate         string `json:"total_amount_rate"`
	Remark                  string `json:"remark"`
}

// StallTurnoverInput 是档口每月卡机、汇来米、成本和保底基数的录入数据。
type StallTurnoverInput struct {
	StallID          uint64 `json:"stall_id"`
	Month            string `json:"month"`
	SchoolAmount     string `json:"school_amount"`
	CardAmount       string `json:"card_amount"`
	HuilaimiAmount   string `json:"huilaimi_amount"`
	CostAmount       string `json:"cost_amount"`
	GuaranteedAmount string `json:"guaranteed_amount"`
	// ManualOwnCommission 留空时按档口规则计算；填写后仅覆盖当前档口当月的我方抽成。
	ManualOwnCommission string `json:"manual_own_commission"`
	Remark              string `json:"remark"`
}

// StallSettingsInput 是指定月份的康莱德公司抽成设置。
type StallSettingsInput struct {
	Month       string `json:"month"`
	CanteenName string `json:"canteen_name"`
	ConradRate  string `json:"conrad_rate"`
	Remark      string `json:"remark"`
}

// StallMonthlyRuleInput 是指定档口指定月份的规则覆盖，不会修改档口长期资料。
type StallMonthlyRuleInput struct {
	StallID                 uint64 `json:"stall_id"`
	Month                   string `json:"month"`
	MonthlyRent             string `json:"monthly_rent"`
	FixedCommissionAmount   string `json:"fixed_commission_amount"`
	DefaultGuaranteedAmount string `json:"default_guaranteed_amount"`
	AllocationMode          string `json:"allocation_mode"`
	ProfitJiaRate           string `json:"profit_jia_rate"`
	ProfitYiRate            string `json:"profit_yi_rate"`
	GuaranteedRate          string `json:"guaranteed_rate"`
	BaseAmountRate          string `json:"base_amount_rate"`
	BaseAmountFactor        string `json:"base_amount_factor"`
	TotalAmountRate         string `json:"total_amount_rate"`
	Remark                  string `json:"remark"`
}

func (v *StallMonthlyRuleInput) Validate() error {
	if v.StallID == 0 {
		return fmt.Errorf("档口不能为空")
	}
	if _, err := time.Parse("2006-01", strings.TrimSpace(v.Month)); err != nil {
		return fmt.Errorf("月份格式不正确")
	}
	v.Month = strings.TrimSpace(v.Month)
	if v.AllocationMode == "" {
		v.AllocationMode = "custom"
	}
	if v.AllocationMode != "custom" && v.AllocationMode != "profit_share" && v.AllocationMode != "none" {
		return fmt.Errorf("抽成模式不正确")
	}
	// 甲乙分成是全局固定口径，不由单个档口或当月规则配置。
	v.ProfitJiaRate, v.ProfitYiRate = "90", "10"
	if len([]rune(v.Remark)) > 500 {
		return fmt.Errorf("备注不能超过500字")
	}
	for _, item := range []struct {
		value, label string
	}{{v.MonthlyRent, "月租"}, {v.FixedCommissionAmount, "固定抽成金额"}, {v.DefaultGuaranteedAmount, "保底金额"}} {
		if err := validateDecimal(item.value, item.label, 100000000); err != nil {
			return err
		}
	}
	// 月度规则中的金额列为非空 DECIMAL，留空代表未设置，按0保存。
	if strings.TrimSpace(v.MonthlyRent) == "" {
		v.MonthlyRent = "0"
	}
	if strings.TrimSpace(v.FixedCommissionAmount) == "" {
		v.FixedCommissionAmount = "0"
	}
	if strings.TrimSpace(v.DefaultGuaranteedAmount) == "" {
		v.DefaultGuaranteedAmount = "0"
	}
	for _, item := range []*struct {
		value *string
		label string
	}{{&v.ProfitJiaRate, "甲分成比例"}, {&v.ProfitYiRate, "乙分成比例"}, {&v.GuaranteedRate, "保底抽成比例"}, {&v.BaseAmountRate, "营业额第一层比例"}, {&v.BaseAmountFactor, "营业额第二层比例"}, {&v.TotalAmountRate, "总营业额比例"}} {
		if err := normalizePercentage(item.value, item.label); err != nil {
			return err
		}
	}
	return nil
}

// StallMonthlyExpenseInput 保存指定月份的水电；人工成本始终从工资明细实发金额汇总。
type StallMonthlyExpenseInput struct {
	Month             string `json:"month"`
	WaterAmount       string `json:"water_amount"`
	ElectricityAmount string `json:"electricity_amount"`
}

// StallMonthlyExpenseLedgerInput 确认月度核算产生的水电、康莱德及营业收支流水。
type StallMonthlyExpenseLedgerInput struct {
	Month      string `json:"month"`
	AccountID  uint64 `json:"account_id"`
	OccurredOn string `json:"occurred_on"`
}

// StallMonthlyExpenseLedgerRevokeInput 撤销指定月份由月度核算自动生成的流水。
type StallMonthlyExpenseLedgerRevokeInput struct {
	Month string `json:"month"`
}

// StallCostInput 是档口月度成本；非自营成本仅记录和入账，不参与甲乙分配。
type StallCostInput struct {
	StallID         uint64 `json:"stall_id"`
	Month           string `json:"month"`
	CostType        string `json:"cost_type"`
	Amount          string `json:"amount"`
	Remark          string `json:"remark"`
	AllSelfOperated bool   `json:"all_self_operated"`
}

// StallCostLedgerInput 将一条自营档口成本写入台账支出。
type StallCostLedgerInput struct {
	CostID     uint64 `json:"cost_id"`
	AccountID  uint64 `json:"account_id"`
	OccurredOn string `json:"occurred_on"`
}

func (v *StallCostLedgerInput) Validate() error {
	if v.CostID == 0 || v.AccountID == 0 {
		return fmt.Errorf("成本和账户不能为空")
	}
	if _, err := time.Parse("2006-01-02", strings.TrimSpace(v.OccurredOn)); err != nil {
		return fmt.Errorf("入账日期格式不正确")
	}
	return nil
}

func (v *StallCostInput) Validate() error {
	if v.StallID == 0 && !v.AllSelfOperated {
		return fmt.Errorf("档口不能为空")
	}
	if _, err := time.Parse("2006-01", strings.TrimSpace(v.Month)); err != nil {
		return fmt.Errorf("成本月份格式必须是YYYY-MM")
	}
	v.Month = strings.TrimSpace(v.Month)
	v.CostType = strings.TrimSpace(v.CostType)
	if v.CostType == "" {
		v.CostType = "other"
	}
	switch v.CostType {
	case "ingredient", "water", "electricity", "labor", "other":
	default:
		return fmt.Errorf("成本类型不正确")
	}
	if v.AllSelfOperated && v.CostType != "water" && v.CostType != "electricity" {
		return fmt.Errorf("全部自营档口仅可登记水费或电费")
	}
	if err := validateDecimal(v.Amount, "成本金额", 100000000); err != nil {
		return err
	}
	if strings.TrimSpace(v.Amount) == "" {
		return fmt.Errorf("成本金额不能为空")
	}
	if len([]rune(v.Remark)) > 255 {
		return fmt.Errorf("备注不能超过255字")
	}
	return nil
}

func validateDecimal(value, label string, max float64) error {
	value = strings.TrimSpace(value)
	if value == "" {
		return nil
	}
	n, err := strconv.ParseFloat(value, 64)
	if err != nil || n < 0 || n > max {
		return fmt.Errorf("%s格式不正确", label)
	}
	return nil
}

// normalizePercentage 将页面输入的百分数转换为数据库使用的小数比例。
func normalizePercentage(value *string, label string) error {
	text := strings.TrimSpace(*value)
	if text == "" {
		*value = "0"
		return nil
	}
	n, err := strconv.ParseFloat(text, 64)
	if err != nil || n < 0 || n > 100 {
		return fmt.Errorf("%s格式不正确，请填写0到100之间的百分数", label)
	}
	*value = strconv.FormatFloat(n/100, 'f', 8, 64)
	return nil
}

func (v *StallInput) Validate() error {
	v.StallNo = strings.TrimSpace(v.StallNo)
	v.Name = strings.TrimSpace(v.Name)
	v.ResponsibleName = strings.TrimSpace(v.ResponsibleName)
	v.ResponsibleMobile = strings.TrimSpace(v.ResponsibleMobile)
	if v.StallNo == "" || len([]rune(v.StallNo)) > 64 {
		return fmt.Errorf("档口编号不能为空且不能超过64字")
	}
	if v.Name == "" || len([]rune(v.Name)) > 128 {
		return fmt.Errorf("档口名称不能为空且不能超过128字")
	}
	if len([]rune(v.ResponsibleName)) > 64 || len([]rune(v.ResponsibleMobile)) > 32 || len([]rune(v.Remark)) > 500 {
		return fmt.Errorf("档口资料长度不正确")
	}
	if v.AllocationMode == "" {
		v.AllocationMode = "custom"
	}
	if v.AllocationMode != "custom" && v.AllocationMode != "profit_share" && v.AllocationMode != "none" {
		return fmt.Errorf("抽成模式不正确")
	}
	// 甲乙分成是全局固定口径，不由单个档口长期资料配置。
	v.ProfitJiaRate, v.ProfitYiRate = "90", "10"
	for _, item := range []struct {
		value, label string
	}{{v.MonthlyRent, "月租"}, {v.FixedCommissionAmount, "固定抽成金额"}, {v.DefaultGuaranteedAmount, "默认保底金额"}} {
		if err := validateDecimal(item.value, item.label, 100000000); err != nil {
			return err
		}
	}
	// 档口长期规则的金额列同样统一归零，避免浏览器留空时写入空字符串。
	if strings.TrimSpace(v.MonthlyRent) == "" {
		v.MonthlyRent = "0"
	}
	if strings.TrimSpace(v.FixedCommissionAmount) == "" {
		v.FixedCommissionAmount = "0"
	}
	if strings.TrimSpace(v.DefaultGuaranteedAmount) == "" {
		v.DefaultGuaranteedAmount = "0"
	}
	for _, item := range []*struct {
		value *string
		label string
	}{{&v.ProfitJiaRate, "甲分成比例"}, {&v.ProfitYiRate, "乙分成比例"}, {&v.GuaranteedRate, "保底抽成比例"}, {&v.BaseAmountRate, "营业额第一层比例"}, {&v.BaseAmountFactor, "营业额第二层比例"}, {&v.TotalAmountRate, "总营业额比例"}} {
		if err := normalizePercentage(item.value, item.label); err != nil {
			return err
		}
	}
	return nil
}

func (v *StallTurnoverInput) Validate() error {
	if v.StallID == 0 {
		return fmt.Errorf("档口不能为空")
	}
	if _, err := time.Parse("2006-01", strings.TrimSpace(v.Month)); err != nil {
		return fmt.Errorf("营业额月份格式必须是YYYY-MM")
	}
	for _, item := range []struct{ value, label string }{{v.CardAmount, "学校卡机营业额"}, {v.HuilaimiAmount, "汇来米营业额"}, {v.CostAmount, "成本"}, {v.GuaranteedAmount, "保底金额"}, {v.ManualOwnCommission, "手动我方抽成"}} {
		if err := validateDecimal(item.value, item.label, 1000000000); err != nil {
			return err
		}
	}
	if strings.TrimSpace(v.SchoolAmount) == "" {
		v.SchoolAmount = "0"
	}
	if strings.TrimSpace(v.CardAmount) == "" {
		v.CardAmount = "0"
	}
	if strings.TrimSpace(v.HuilaimiAmount) == "" {
		v.HuilaimiAmount = "0"
	}
	if strings.TrimSpace(v.CostAmount) == "" {
		v.CostAmount = "0"
	}
	if strings.TrimSpace(v.GuaranteedAmount) == "" {
		v.GuaranteedAmount = "0"
	}
	if len([]rune(v.Remark)) > 500 {
		return fmt.Errorf("备注不能超过500字")
	}
	return nil
}

func (v *StallSettingsInput) Validate() error {
	if _, err := time.Parse("2006-01", strings.TrimSpace(v.Month)); err != nil {
		return fmt.Errorf("抽成月份格式必须是YYYY-MM")
	}
	if v.CanteenName == "" {
		v.CanteenName = "厦门理工知行2食堂"
	}
	if len([]rune(v.CanteenName)) > 128 || len([]rune(v.Remark)) > 500 {
		return fmt.Errorf("食堂设置内容过长")
	}
	if err := normalizePercentage(&v.ConradRate, "康莱德抽成比例"); err != nil {
		return err
	}
	return nil
}

func (v *StallMonthlyExpenseInput) Validate() error {
	if _, err := time.Parse("2006-01", strings.TrimSpace(v.Month)); err != nil {
		return fmt.Errorf("经营支出月份格式必须是YYYY-MM")
	}
	v.Month = strings.TrimSpace(v.Month)
	for _, item := range []struct{ value, label string }{{v.WaterAmount, "水费"}, {v.ElectricityAmount, "电费"}} {
		if err := validateDecimal(item.value, item.label, 1000000000); err != nil {
			return err
		}
	}
	// 水费和电费是非空金额列；页面留空代表本月未发生，统一写入0。
	if strings.TrimSpace(v.WaterAmount) == "" {
		v.WaterAmount = "0"
	}
	if strings.TrimSpace(v.ElectricityAmount) == "" {
		v.ElectricityAmount = "0"
	}
	return nil
}

func (v *StallMonthlyExpenseLedgerInput) Validate() error {
	if _, err := time.Parse("2006-01", strings.TrimSpace(v.Month)); err != nil {
		return fmt.Errorf("经营支出月份格式必须是YYYY-MM")
	}
	v.Month = strings.TrimSpace(v.Month)
	if v.OccurredOn == "" {
		v.OccurredOn = v.Month + "-01"
	}
	occurredOn, err := time.Parse("2006-01-02", v.OccurredOn)
	if err != nil {
		return fmt.Errorf("入账日期格式必须是YYYY-MM-DD")
	}
	// 月度核算只能写入所选月份，避免确认历史月份时误把流水记到其他月份。
	if occurredOn.Format("2006-01") != v.Month {
		return fmt.Errorf("入账日期必须属于核算月份%s", v.Month)
	}
	return nil
}

func (v *StallMonthlyExpenseLedgerRevokeInput) Validate() error {
	v.Month = strings.TrimSpace(v.Month)
	if _, err := time.Parse("2006-01", v.Month); err != nil {
		return fmt.Errorf("经营支出月份格式必须是YYYY-MM")
	}
	return nil
}

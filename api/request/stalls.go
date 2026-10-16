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
	Remark           string `json:"remark"`
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
	if len([]rune(v.Remark)) > 500 {
		return fmt.Errorf("备注不能超过500字")
	}
	for _, item := range []struct {
		value, label string
	}{{v.MonthlyRent, "月租"}, {v.DefaultGuaranteedAmount, "保底金额"}} {
		if err := validateDecimal(item.value, item.label, 100000000); err != nil {
			return err
		}
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

// StallLedgerInput 是把档口当月抽成写入收入台账所需的信息。
type StallLedgerInput struct {
	StallID    uint64 `json:"stall_id"`
	Month      string `json:"month"`
	AccountID  uint64 `json:"account_id"`
	OccurredOn string `json:"occurred_on"`
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
	for _, item := range []struct {
		value, label string
	}{{v.MonthlyRent, "月租"}, {v.DefaultGuaranteedAmount, "默认保底金额"}} {
		if err := validateDecimal(item.value, item.label, 100000000); err != nil {
			return err
		}
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
	for _, item := range []struct{ value, label string }{{v.CardAmount, "学校卡机营业额"}, {v.HuilaimiAmount, "汇来米营业额"}, {v.CostAmount, "成本"}, {v.GuaranteedAmount, "保底金额"}} {
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
	return normalizePercentage(&v.ConradRate, "康莱德抽成比例")
}

func (v *StallLedgerInput) Validate() error {
	if v.StallID == 0 || v.AccountID == 0 {
		return fmt.Errorf("档口和入账账户不能为空")
	}
	if _, err := time.Parse("2006-01", strings.TrimSpace(v.Month)); err != nil {
		return fmt.Errorf("营业额月份格式必须是YYYY-MM")
	}
	if v.OccurredOn == "" {
		v.OccurredOn = strings.TrimSpace(v.Month) + "-01"
	}
	if _, err := time.Parse("2006-01-02", v.OccurredOn); err != nil {
		return fmt.Errorf("入账日期格式必须是YYYY-MM-DD")
	}
	return nil
}

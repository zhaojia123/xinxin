package request

import (
	"fmt"
	"strconv"
	"strings"
)

// PurchaseInput 是每日向供货商采购的一行明细。
type PurchaseInput struct {
	Date         string `json:"date"`
	SupplierName string `json:"supplier_name"`
	Category     string `json:"category"`
	ProductName  string `json:"product_name"`
	Quantity     string `json:"quantity"`
	Unit         string `json:"unit"`
	UnitPrice    string `json:"unit_price"`
	Remark       string `json:"remark"`
}

func (v *PurchaseInput) Validate() error {
	v.SupplierName = strings.TrimSpace(v.SupplierName)
	v.Category = strings.TrimSpace(v.Category)
	v.ProductName = strings.TrimSpace(v.ProductName)
	v.Unit = strings.TrimSpace(v.Unit)
	v.Remark = strings.TrimSpace(v.Remark)
	if !validDate(v.Date, false) {
		return fmt.Errorf("请选择正确的采购日期")
	}
	if v.SupplierName == "" || !textLength(v.SupplierName, 64) {
		return fmt.Errorf("供货商名称必填且不能超过64字")
	}
	if v.Category == "" || !textLength(v.Category, 32) {
		return fmt.Errorf("采购品类必填且不能超过32字")
	}
	if v.ProductName == "" || !textLength(v.ProductName, 128) {
		return fmt.Errorf("菜品或物料名称必填且不能超过128字")
	}
	if v.Unit == "" || !textLength(v.Unit, 16) {
		return fmt.Errorf("采购单位必填且不能超过16字")
	}
	quantity, err := strconv.ParseFloat(v.Quantity, 64)
	if err != nil || quantity <= 0 || len(strings.Split(v.Quantity, ".")[0]) > 10 {
		return fmt.Errorf("采购数量必须大于0")
	}
	if err := ValidateMoney(v.UnitPrice, false, 10); err != nil {
		return fmt.Errorf("单价格式不正确")
	}
	if !textLength(v.Remark, 500) {
		return fmt.Errorf("备注不能超过500字")
	}
	return nil
}

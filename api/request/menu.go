package request

import (
	"fmt"
	"strconv"
	"strings"
)

// MenuItemInput 是菜单新增、编辑和行内修改共用的数据结构。
type MenuItemInput struct {
	Restaurant    string `json:"restaurant"`
	WindowName    string `json:"window_name"`
	Name          string `json:"name"`
	Specification string `json:"specification"`
	Price         string `json:"price"`
	PriceText     string `json:"price_text"`
	SalesPeriod   string `json:"sales_period"`
	PriceRange    string `json:"price_range"`
	PriceCategory string `json:"price_category"`
	ItemCategory  string `json:"item_category"`
	Remark        string `json:"remark"`
}

func (v *MenuItemInput) Validate() error {
	v.Restaurant = strings.TrimSpace(v.Restaurant)
	v.WindowName = strings.TrimSpace(v.WindowName)
	v.Name = strings.TrimSpace(v.Name)
	v.Specification = strings.TrimSpace(v.Specification)
	v.Price = strings.TrimSpace(v.Price)
	v.PriceText = strings.TrimSpace(v.PriceText)
	v.SalesPeriod = strings.TrimSpace(v.SalesPeriod)
	v.PriceRange = strings.TrimSpace(v.PriceRange)
	v.PriceCategory = strings.TrimSpace(v.PriceCategory)
	v.ItemCategory = strings.TrimSpace(v.ItemCategory)
	v.Remark = strings.TrimSpace(v.Remark)
	if v.Name == "" {
		return fmt.Errorf("菜品名称不能为空")
	}
	if len([]rune(v.Name)) > 128 || len([]rune(v.WindowName)) > 128 || len([]rune(v.Restaurant)) > 128 || len([]rune(v.Specification)) > 64 || len([]rune(v.PriceText)) > 32 || len([]rune(v.SalesPeriod)) > 128 || len([]rune(v.PriceRange)) > 64 || len([]rune(v.PriceCategory)) > 64 || len([]rune(v.ItemCategory)) > 64 || len([]rune(v.Remark)) > 500 {
		return fmt.Errorf("菜单字段长度不正确")
	}
	if v.Price == "" {
		v.Price = "0"
	}
	price, err := strconv.ParseFloat(v.Price, 64)
	if err != nil || price < 0 || price > 1000000 {
		return fmt.Errorf("价格格式不正确")
	}
	return nil
}

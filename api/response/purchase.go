package response

type PurchaseItem struct {
	ID           uint64 `json:"id"`
	Date         string `json:"date"`
	SupplierName string `json:"supplier_name"`
	Category     string `json:"category"`
	ProductName  string `json:"product_name"`
	Quantity     string `json:"quantity"`
	Unit         string `json:"unit"`
	UnitPrice    string `json:"unit_price"`
	TotalPrice   string `json:"total_price"`
	Remark       string `json:"remark"`
	ChangeDetail string `json:"change_detail"`
	ChangedAt    string `json:"changed_at"`
	Modified     bool   `json:"modified"`
}

type PurchaseCategoryTotal struct {
	Category   string `json:"category"`
	TotalPrice string `json:"total_price"`
}

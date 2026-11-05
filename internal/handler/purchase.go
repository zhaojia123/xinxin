package handler

import (
	"net/http"
	"time"

	"friends-records/api/request"
	"friends-records/api/response"
	"friends-records/internal/httpx"
	"friends-records/internal/models/mysql"
	"friends-records/internal/token"
)

type PurchasePageData struct {
	ActiveMenu string
	Items      []response.PurchaseItem
	Totals     []response.PurchaseCategoryTotal
	StartDate  string
	EndDate    string
}

func purchaseDateRange(r *http.Request) (string, string, error) {
	start, end := r.URL.Query().Get("start_date"), r.URL.Query().Get("end_date")
	if start == "" && end == "" {
		start = time.Now().Format("2006-01-02")
	}
	return mysql.NormalizeDateRange(start, end)
}

func (h *Handler) PurchasesPage(w http.ResponseWriter, r *http.Request) {
	if !getOnly(w, r) {
		return
	}
	start, end, err := purchaseDateRange(r)
	if err != nil {
		httpx.Error(w, http.StatusBadRequest, err.Error())
		return
	}
	items, err := h.Store.PurchaseItems(r.Context(), start, end)
	if err != nil {
		fail(w, err, "采购清单读取失败")
		return
	}
	totals, err := h.Store.PurchaseCategoryTotals(r.Context(), start, end)
	if err != nil {
		fail(w, err, "采购分类统计失败")
		return
	}
	h.render(w, "purchases.html", PurchasePageData{ActiveMenu: "purchases", Items: items, Totals: totals, StartDate: start, EndDate: end})
}

func (h *Handler) AdminPurchasesAPI(tokens *token.Manager) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		id, ok := adminID(w, r, false)
		if !ok {
			return
		}
		switch r.Method {
		case http.MethodGet:
			if id != 0 {
				item, err := h.Store.PurchaseItem(r.Context(), id)
				if err != nil {
					adminWriteError(w, err)
					return
				}
				httpx.JSON(w, http.StatusOK, item)
				return
			}
			start, end, err := purchaseDateRange(r)
			if err != nil {
				httpx.Error(w, http.StatusBadRequest, err.Error())
				return
			}
			items, err := h.Store.PurchaseItems(r.Context(), start, end)
			if err != nil {
				fail(w, err, "采购清单读取失败")
				return
			}
			totals, err := h.Store.PurchaseCategoryTotals(r.Context(), start, end)
			if err != nil {
				fail(w, err, "采购分类统计失败")
				return
			}
			httpx.JSON(w, http.StatusOK, map[string]any{"records": items, "totals": totals})
		case http.MethodPost, http.MethodPut:
			actor, ok := adminActor(w, r, tokens)
			if !ok {
				return
			}
			if r.Method == http.MethodPost && id != 0 {
				httpx.Error(w, http.StatusBadRequest, "新增采购明细不能携带ID")
				return
			}
			var input request.PurchaseInput
			if !decodeJSON(w, r, &input) {
				return
			}
			if err := input.Validate(); err != nil {
				httpx.Error(w, http.StatusBadRequest, err.Error())
				return
			}
			id, err := h.Store.SavePurchaseItem(r.Context(), id, actor, input)
			if err != nil {
				adminWriteError(w, err)
				return
			}
			httpx.JSON(w, createdStatus(r.Method), map[string]any{"id": id})
		case http.MethodDelete:
			actor, ok := adminActor(w, r, tokens)
			if !ok {
				return
			}
			if id == 0 {
				httpx.Error(w, http.StatusBadRequest, "删除采购明细必须提供ID")
				return
			}
			if err := h.Store.DeletePurchaseItem(r.Context(), id, actor); err != nil {
				adminWriteError(w, err)
				return
			}
			httpx.JSON(w, http.StatusOK, map[string]any{"id": id})
		default:
			httpx.MethodNotAllowed(w, http.MethodGet, http.MethodPost, http.MethodPut, http.MethodDelete)
		}
	}
}

func (h *Handler) MiniPurchasesAPI(w http.ResponseWriter, r *http.Request) {
	id, ok := miniID(w, r)
	if !ok {
		return
	}
	switch r.Method {
	case http.MethodGet:
		if id != 0 {
			item, err := h.Store.PurchaseItem(r.Context(), id)
			if err != nil {
				miniFail(w, err)
				return
			}
			httpx.JSON(w, http.StatusOK, item)
			return
		}
		start, end, err := purchaseDateRange(r)
		if err != nil {
			httpx.Error(w, http.StatusBadRequest, err.Error())
			return
		}
		items, err := h.Store.PurchaseItems(r.Context(), start, end)
		if err != nil {
			miniFail(w, err)
			return
		}
		totals, err := h.Store.PurchaseCategoryTotals(r.Context(), start, end)
		if err != nil {
			miniFail(w, err)
			return
		}
		httpx.JSON(w, http.StatusOK, map[string]any{"records": items, "totals": totals})
	case http.MethodPost, http.MethodPut:
		if r.Method == http.MethodPost && id != 0 {
			httpx.Error(w, http.StatusBadRequest, "新增采购明细不能携带ID")
			return
		}
		var input request.PurchaseInput
		if !decodeJSON(w, r, &input) {
			return
		}
		if err := input.Validate(); err != nil {
			httpx.Error(w, http.StatusBadRequest, err.Error())
			return
		}
		id, err := h.Store.SavePurchaseItem(r.Context(), id, miniActor(r), input)
		if err != nil {
			miniFail(w, err)
			return
		}
		httpx.JSON(w, createdStatus(r.Method), map[string]any{"id": id})
	case http.MethodDelete:
		if id == 0 {
			httpx.Error(w, http.StatusBadRequest, "删除采购明细必须提供ID")
			return
		}
		if err := h.Store.DeletePurchaseItem(r.Context(), id, miniActor(r)); err != nil {
			miniFail(w, err)
			return
		}
		httpx.JSON(w, http.StatusOK, map[string]any{"id": id})
	default:
		httpx.MethodNotAllowed(w, http.MethodGet, http.MethodPost, http.MethodPut, http.MethodDelete)
	}
}

// PurchaseExport 导出标准 XLSX，保证 Excel 与 WPS 都能直接打开。
func (h *Handler) PurchaseExport(tokens *token.Manager) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if _, ok := adminActor(w, r, tokens); !ok {
			return
		}
		if r.Method != http.MethodGet {
			httpx.MethodNotAllowed(w, http.MethodGet)
			return
		}
		start, end, err := purchaseDateRange(r)
		if err != nil {
			httpx.Error(w, http.StatusBadRequest, err.Error())
			return
		}
		items, err := h.Store.PurchaseItems(r.Context(), start, end)
		if err != nil {
			fail(w, err, "采购清单导出失败")
			return
		}
		writePurchaseWorkbook(w, items, start, end)
	}
}

func (h *Handler) MiniPurchaseExport(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		httpx.MethodNotAllowed(w, http.MethodGet)
		return
	}
	start, end, err := purchaseDateRange(r)
	if err != nil {
		httpx.Error(w, http.StatusBadRequest, err.Error())
		return
	}
	items, err := h.Store.PurchaseItems(r.Context(), start, end)
	if err != nil {
		miniFail(w, err)
		return
	}
	writePurchaseWorkbook(w, items, start, end)
}

func writePurchaseWorkbook(w http.ResponseWriter, items []response.PurchaseItem, start, end string) {
	records := make([][]exportCell, 0, len(items))
	for _, item := range items {
		records = append(records, []exportCell{exportText(item.Date), exportText(item.SupplierName), exportText(item.Category), exportText(item.ProductName), exportNumber(item.Quantity), exportText(item.Unit), exportNumber(item.UnitPrice), exportNumber(item.TotalPrice), exportText(item.Remark)})
	}
	if err := writeExcelXLSX(w, "采购清单_"+start+"_至_"+end+".xlsx", "采购清单（"+start+" 至 "+end+"）", "采购清单", []string{"日期", "供货商", "品类", "菜品/物料", "数量", "单位", "单价（元）", "总价（元）", "备注"}, records, []int{90, 120, 80, 160, 70, 70, 85, 90, 180}); err != nil {
		fail(w, err, "采购清单导出失败")
	}
}

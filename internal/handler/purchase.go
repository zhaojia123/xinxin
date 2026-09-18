package handler

import (
	"fmt"
	"html"
	"net/http"
	"net/url"
	"strings"
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

// PurchaseExport 导出 Excel 可直接打开的 .xls 文件，避免引入重量级第三方依赖。
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
	filename := url.QueryEscape("采购清单_" + start + "_至_" + end + ".xls")
	w.Header().Set("Content-Type", "application/vnd.ms-excel; charset=utf-8")
	w.Header().Set("Content-Disposition", "attachment; filename*=UTF-8''"+filename)
	w.WriteHeader(http.StatusOK)
	_, _ = w.Write([]byte(purchaseWorkbook(items, start, end)))
}

func purchaseWorkbook(items []response.PurchaseItem, start, end string) string {
	var b strings.Builder
	b.WriteString("<?xml version=\"1.0\" encoding=\"UTF-8\"?><?mso-application progid=\"Excel.Sheet\"?><Workbook xmlns=\"urn:schemas-microsoft-com:office:spreadsheet\" xmlns:ss=\"urn:schemas-microsoft-com:office:spreadsheet\"><Worksheet ss:Name=\"采购清单\"><Table>")
	b.WriteString("<Row><Cell><Data ss:Type=\"String\">采购清单")
	b.WriteString(html.EscapeString("（" + start + " 至 " + end + "）"))
	b.WriteString("</Data></Cell></Row><Row>")
	for _, header := range []string{"日期", "供货商", "品类", "菜品/物料", "数量", "单位", "单价（元）", "总价（元）", "备注"} {
		fmt.Fprintf(&b, "<Cell><Data ss:Type=\"String\">%s</Data></Cell>", html.EscapeString(header))
	}
	b.WriteString("</Row>")
	for _, item := range items {
		b.WriteString("<Row>")
		for _, value := range []string{item.Date, item.SupplierName, item.Category, item.ProductName, item.Quantity, item.Unit, item.UnitPrice, item.TotalPrice, item.Remark} {
			fmt.Fprintf(&b, "<Cell><Data ss:Type=\"String\">%s</Data></Cell>", html.EscapeString(value))
		}
		b.WriteString("</Row>")
	}
	b.WriteString("</Table></Worksheet></Workbook>")
	return b.String()
}

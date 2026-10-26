package handler

import (
	"fmt"
	"friends-records/api/response"
	"friends-records/internal/httpx"
	"friends-records/internal/models/mysql"
	"net/http"
	"strconv"
	"time"
)

type LedgerPageData struct {
	ActiveMenu  string
	Records     []response.LedgerRecord
	Summary     response.LedgerSummary
	Trend       []response.TrendMonth
	Year        int
	FilterMonth string
	CategoryID  uint64
	Categories  []mysql.MiniOption
}

func ledgerDateRange(r *http.Request) (string, string, error) {
	query := r.URL.Query()
	if query.Get("all") == "1" {
		return "1000-01-01", "9998-12-31", nil
	}
	start, end := query.Get("start_date"), query.Get("end_date")
	if start == "" && end == "" && query.Get("month") != "" {
		start, end = mysql.MonthDateRange(query.Get("month"))
	}
	if start == "" && end == "" {
		start, end = mysql.MonthDateRange(time.Now().Format("2006-01"))
		return start, end, nil
	}
	return mysql.NormalizeDateRange(start, end)
}

func ledgerAllHistory(start, end string) bool {
	return start == "1000-01-01" && end == "9998-12-31"
}

func ledgerCategoryID(r *http.Request) (uint64, error) {
	value := r.URL.Query().Get("category_id")
	if value == "" {
		return 0, nil
	}
	id, err := strconv.ParseUint(value, 10, 64)
	if err != nil {
		return 0, fmt.Errorf("收支分类参数不正确")
	}
	return id, nil
}

func (h *Handler) LedgerPage(w http.ResponseWriter, r *http.Request) {
	if !getOnly(w, r) {
		return
	}
	start, end, err := ledgerDateRange(r)
	if err != nil {
		httpx.Error(w, http.StatusBadRequest, err.Error())
		return
	}
	categoryID, err := ledgerCategoryID(r)
	if err != nil {
		httpx.Error(w, http.StatusBadRequest, err.Error())
		return
	}
	records, err := h.Store.LedgerRangeFiltered(r.Context(), start, end, categoryID)
	if err != nil {
		fail(w, err, "台账流水读取失败")
		return
	}
	summary, err := h.Store.LedgerSummaryRangeFiltered(r.Context(), start, end, categoryID)
	if err != nil {
		fail(w, err, "台账统计读取失败")
		return
	}
	year := time.Now().Year()
	if !ledgerAllHistory(start, end) {
		year = mysql.LedgerYear(start[:7])
	}
	trend, err := h.Store.LedgerTrendFiltered(r.Context(), year, categoryID)
	if err != nil {
		fail(w, err, "年度趋势读取失败")
		return
	}
	options, err := h.Store.MiniOptions(r.Context())
	if err != nil {
		fail(w, err, "台账筛选项读取失败")
		return
	}
	categories := options["categories"]
	filterMonth := r.URL.Query().Get("month")
	if filterMonth == "" && !ledgerAllHistory(start, end) && r.URL.Query().Get("start_date") == "" && r.URL.Query().Get("end_date") == "" {
		filterMonth = time.Now().Format("2006-01")
	}
	h.render(w, "ledger.html", LedgerPageData{ActiveMenu: "ledger", Records: records, Summary: summary, Trend: trend, Year: year, FilterMonth: filterMonth, CategoryID: categoryID, Categories: categories})
}
func (h *Handler) LedgerAPI(w http.ResponseWriter, r *http.Request) {
	if !getOnly(w, r) {
		return
	}
	start, end, err := ledgerDateRange(r)
	if err != nil {
		httpx.Error(w, http.StatusBadRequest, err.Error())
		return
	}
	categoryID, err := ledgerCategoryID(r)
	if err != nil {
		httpx.Error(w, http.StatusBadRequest, err.Error())
		return
	}
	records, err := h.Store.LedgerRangeFiltered(r.Context(), start, end, categoryID)
	if err != nil {
		fail(w, err, "台账流水读取失败")
		return
	}
	summary, err := h.Store.LedgerSummaryRangeFiltered(r.Context(), start, end, categoryID)
	if err != nil {
		fail(w, err, "台账统计读取失败")
		return
	}
	year := time.Now().Year()
	if !ledgerAllHistory(start, end) {
		year = mysql.LedgerYear(start[:7])
	}
	trend, err := h.Store.LedgerTrendFiltered(r.Context(), year, categoryID)
	if err != nil {
		fail(w, err, "年度趋势读取失败")
		return
	}
	httpx.JSON(w, http.StatusOK, map[string]any{"records": records, "summary": summary, "trend": trend, "category_id": categoryID})
}

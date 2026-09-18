package handler

import (
	"friends-records/api/response"
	"friends-records/internal/httpx"
	"friends-records/internal/models/mysql"
	"net/http"
)

type LedgerPageData struct {
	ActiveMenu string
	Records    []response.LedgerRecord
	Summary    response.LedgerSummary
	Trend      []response.TrendMonth
	Year       int
	StartDate  string
	EndDate    string
}

func ledgerDateRange(r *http.Request) (string, string, error) {
	query := r.URL.Query()
	start, end := query.Get("start_date"), query.Get("end_date")
	if start == "" && end == "" && query.Get("month") != "" {
		start, end = mysql.MonthDateRange(query.Get("month"))
	}
	return mysql.NormalizeDateRange(start, end)
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
	records, err := h.Store.LedgerRange(r.Context(), start, end)
	if err != nil {
		fail(w, err, "台账流水读取失败")
		return
	}
	summary, err := h.Store.LedgerSummaryRange(r.Context(), start, end)
	if err != nil {
		fail(w, err, "台账统计读取失败")
		return
	}
	year := mysql.LedgerYear(start[:7])
	trend, err := h.Store.LedgerTrend(r.Context(), year)
	if err != nil {
		fail(w, err, "年度趋势读取失败")
		return
	}
	h.render(w, "ledger.html", LedgerPageData{ActiveMenu: "ledger", Records: records, Summary: summary, Trend: trend, Year: year, StartDate: start, EndDate: end})
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
	records, err := h.Store.LedgerRange(r.Context(), start, end)
	if err != nil {
		fail(w, err, "台账流水读取失败")
		return
	}
	summary, err := h.Store.LedgerSummaryRange(r.Context(), start, end)
	if err != nil {
		fail(w, err, "台账统计读取失败")
		return
	}
	trend, err := h.Store.LedgerTrend(r.Context(), mysql.LedgerYear(start[:7]))
	if err != nil {
		fail(w, err, "年度趋势读取失败")
		return
	}
	httpx.JSON(w, http.StatusOK, map[string]any{"records": records, "summary": summary, "trend": trend})
}

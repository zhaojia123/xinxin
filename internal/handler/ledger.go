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
}

func (h *Handler) LedgerPage(w http.ResponseWriter, r *http.Request) {
	if !getOnly(w, r) {
		return
	}
	month := mysql.NormalizeMonth(r.URL.Query().Get("month"))
	records, err := h.Store.Ledger(r.Context(), month)
	if err != nil {
		fail(w, err, "台账流水读取失败")
		return
	}
	summary, err := h.Store.LedgerSummary(r.Context(), month)
	if err != nil {
		fail(w, err, "台账统计读取失败")
		return
	}
	year := mysql.LedgerYear(month)
	trend, err := h.Store.LedgerTrend(r.Context(), year)
	if err != nil {
		fail(w, err, "年度趋势读取失败")
		return
	}
	h.render(w, "ledger.html", LedgerPageData{"ledger", records, summary, trend, year})
}
func (h *Handler) LedgerAPI(w http.ResponseWriter, r *http.Request) {
	if !getOnly(w, r) {
		return
	}
	month := mysql.NormalizeMonth(r.URL.Query().Get("month"))
	records, err := h.Store.Ledger(r.Context(), month)
	if err != nil {
		fail(w, err, "台账流水读取失败")
		return
	}
	summary, err := h.Store.LedgerSummary(r.Context(), month)
	if err != nil {
		fail(w, err, "台账统计读取失败")
		return
	}
	trend, err := h.Store.LedgerTrend(r.Context(), mysql.LedgerYear(month))
	if err != nil {
		fail(w, err, "年度趋势读取失败")
		return
	}
	httpx.JSON(w, http.StatusOK, map[string]any{"records": records, "summary": summary, "trend": trend})
}

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

type StallsPageData struct {
	ActiveMenu string
	Keyword    string
	Stalls     []response.Stall
}

type StallMonthlyPageData struct {
	ActiveMenu string
	Month      string
	Keyword    string
	Stalls     []response.Stall
	Summary    response.StallSummary
	Settings   response.StallSettings
	Expenses   response.StallMonthlyExpense
}

type StallCostsPageData struct {
	ActiveMenu string
	Month      string
	Keyword    string
	SelfItems  []response.StallCostEntry
	OtherItems []response.StallCostEntry
	Stalls     []response.Stall
}

func (h *Handler) StallsPage(w http.ResponseWriter, r *http.Request) {
	if !getOnly(w, r) {
		return
	}
	keyword := r.URL.Query().Get("q")
	items, err := h.Store.StallsMaster(r.Context(), keyword)
	if err != nil {
		fail(w, err, "档口数据读取失败")
		return
	}
	h.render(w, "stalls.html", StallsPageData{ActiveMenu: "stalls", Keyword: keyword, Stalls: items})
}

func (h *Handler) StallsMonthlyPage(w http.ResponseWriter, r *http.Request) {
	if !getOnly(w, r) {
		return
	}
	month := r.URL.Query().Get("month")
	if _, err := time.Parse("2006-01", month); err != nil {
		month = time.Now().AddDate(0, -1, 0).Format("2006-01")
	}
	keyword := r.URL.Query().Get("q")
	items, summary, err := h.Store.StallsWithTurnover(r.Context(), month, keyword)
	if err != nil {
		fail(w, err, "档口月度数据读取失败")
		return
	}
	settings, err := h.Store.StallSettings(r.Context(), month)
	if err != nil {
		fail(w, err, "档口抽成设置读取失败")
		return
	}
	expenses, err := h.Store.MonthlyOperatingExpense(r.Context(), month)
	if err != nil {
		fail(w, err, "当月经营支出读取失败")
		return
	}
	mysql.ApplyMonthlyOperatingExpense(&summary, expenses)
	expenses.ConradAmount = summary.ConradCommission
	expenses.CardIncomeAmount = summary.TotalCardTurnover
	expenses.HuilaimiIncomeAmount = summary.TotalHuilaimi
	h.render(w, "stall_settlement.html", StallMonthlyPageData{ActiveMenu: "stall-monthly", Month: month, Keyword: keyword, Stalls: items, Summary: summary, Settings: settings, Expenses: expenses})
}

// StallCostsPage 展示三个自营档口的成本明细，成本会自动汇总回月度核算。
func (h *Handler) StallCostsPage(w http.ResponseWriter, r *http.Request) {
	if !getOnly(w, r) {
		return
	}
	month := r.URL.Query().Get("month")
	if _, err := time.Parse("2006-01", month); err != nil {
		month = time.Now().AddDate(0, -1, 0).Format("2006-01")
	}
	keyword := r.URL.Query().Get("q")
	items, err := h.Store.StallCosts(r.Context(), month, keyword)
	if err != nil {
		fail(w, err, "档口成本读取失败")
		return
	}
	stalls, err := h.Store.CostStalls(r.Context())
	if err != nil {
		fail(w, err, "档口成本选项读取失败")
		return
	}
	selfItems := make([]response.StallCostEntry, 0)
	otherItems := make([]response.StallCostEntry, 0)
	for _, item := range items {
		if item.SelfOperated {
			item.Seq = len(selfItems) + 1
			selfItems = append(selfItems, item)
			continue
		}
		item.Seq = len(otherItems) + 1
		otherItems = append(otherItems, item)
	}
	h.render(w, "stall_costs.html", StallCostsPageData{ActiveMenu: "stall-costs", Month: month, Keyword: keyword, SelfItems: selfItems, OtherItems: otherItems, Stalls: stalls})
}

func (h *Handler) StallsAPI(tokens *token.Manager) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		switch r.Method {
		case http.MethodGet:
			month := r.URL.Query().Get("month")
			if _, err := time.Parse("2006-01", month); err != nil {
				month = time.Now().AddDate(0, -1, 0).Format("2006-01")
			}
			items, summary, err := h.Store.StallsWithTurnover(r.Context(), month, r.URL.Query().Get("q"))
			if err != nil {
				adminWriteError(w, err)
				return
			}
			settings, settingsErr := h.Store.StallSettings(r.Context(), month)
			if settingsErr != nil {
				adminWriteError(w, settingsErr)
				return
			}
			expenses, expenseErr := h.Store.MonthlyOperatingExpense(r.Context(), month)
			if expenseErr != nil {
				adminWriteError(w, expenseErr)
				return
			}
			mysql.ApplyMonthlyOperatingExpense(&summary, expenses)
			expenses.ConradAmount = summary.ConradCommission
			expenses.CardIncomeAmount = summary.TotalCardTurnover
			expenses.HuilaimiIncomeAmount = summary.TotalHuilaimi
			httpx.JSON(w, http.StatusOK, map[string]any{"stalls": items, "summary": summary, "month": month, "settings": settings, "expenses": expenses})
		case http.MethodPost, http.MethodPut:
			id, ok := adminID(w, r, r.Method == http.MethodPut)
			if !ok {
				return
			}
			var input request.StallInput
			if !decodeJSON(w, r, &input) {
				return
			}
			if err := input.Validate(); err != nil {
				httpx.Error(w, http.StatusBadRequest, err.Error())
				return
			}
			actor, ok := adminActor(w, r, tokens)
			if !ok {
				return
			}
			saved, err := h.Store.SaveStall(r.Context(), id, actor, input)
			if err != nil {
				adminWriteError(w, err)
				return
			}
			httpx.JSON(w, createdStatus(r.Method), map[string]any{"id": saved})
		case http.MethodDelete:
			actor, ok := adminActor(w, r, tokens)
			if !ok {
				return
			}
			id, ok := adminID(w, r, true)
			if !ok {
				return
			}
			if err := h.Store.DeleteStall(r.Context(), id, actor); err != nil {
				adminWriteError(w, err)
				return
			}
			httpx.JSON(w, http.StatusOK, map[string]any{"id": id})
		default:
			httpx.MethodNotAllowed(w, http.MethodGet, http.MethodPost, http.MethodPut, http.MethodDelete)
		}
	}
}

// StallSettingsAPI 管理指定月份的康莱德公司抽成比例。
func (h *Handler) StallSettingsAPI(tokens *token.Manager) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost && r.Method != http.MethodPut {
			httpx.MethodNotAllowed(w, http.MethodPost, http.MethodPut)
			return
		}
		actor, ok := adminActor(w, r, tokens)
		if !ok {
			return
		}
		var input request.StallSettingsInput
		if !decodeJSON(w, r, &input) {
			return
		}
		if err := input.Validate(); err != nil {
			httpx.Error(w, http.StatusBadRequest, err.Error())
			return
		}
		if err := h.Store.SaveStallSettings(r.Context(), actor, input); err != nil {
			adminWriteError(w, err)
			return
		}
		httpx.JSON(w, http.StatusOK, map[string]any{"month": input.Month})
	}
}

// StallMonthlyRuleAPI 保存档口指定月份的规则覆盖。
func (h *Handler) StallMonthlyRuleAPI(tokens *token.Manager) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost && r.Method != http.MethodPut {
			httpx.MethodNotAllowed(w, http.MethodPost, http.MethodPut)
			return
		}
		actor, ok := adminActor(w, r, tokens)
		if !ok {
			return
		}
		var input request.StallMonthlyRuleInput
		if !decodeJSON(w, r, &input) {
			return
		}
		if err := input.Validate(); err != nil {
			httpx.Error(w, http.StatusBadRequest, err.Error())
			return
		}
		if err := h.Store.SaveStallMonthlyRule(r.Context(), actor, input); err != nil {
			adminWriteError(w, err)
			return
		}
		httpx.JSON(w, http.StatusOK, map[string]any{"stall_id": input.StallID, "month": input.Month})
	}
}

func (h *Handler) StallTurnoverAPI(tokens *token.Manager) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		switch r.Method {
		case http.MethodPost, http.MethodPut:
			id, ok := adminID(w, r, r.Method == http.MethodPut)
			if !ok {
				return
			}
			var input request.StallTurnoverInput
			if !decodeJSON(w, r, &input) {
				return
			}
			if err := input.Validate(); err != nil {
				httpx.Error(w, http.StatusBadRequest, err.Error())
				return
			}
			actor, ok := adminActor(w, r, tokens)
			if !ok {
				return
			}
			saved, err := h.Store.SaveStallTurnover(r.Context(), id, actor, input)
			if err != nil {
				adminWriteError(w, err)
				return
			}
			httpx.JSON(w, createdStatus(r.Method), map[string]any{"id": saved})
		case http.MethodDelete:
			actor, ok := adminActor(w, r, tokens)
			if !ok {
				return
			}
			id, ok := adminID(w, r, true)
			if !ok {
				return
			}
			if err := h.Store.DeleteStallTurnover(r.Context(), id, actor); err != nil {
				adminWriteError(w, err)
				return
			}
			httpx.JSON(w, http.StatusOK, map[string]any{"id": id})
		default:
			httpx.MethodNotAllowed(w, http.MethodPost, http.MethodPut, http.MethodDelete)
		}
	}
}

// StallCostsAPI 管理三个自营档口的成本，普通档口不在该接口范围内。
func (h *Handler) StallCostsAPI(tokens *token.Manager) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		switch r.Method {
		case http.MethodGet:
			month := r.URL.Query().Get("month")
			if _, err := time.Parse("2006-01", month); err != nil {
				month = time.Now().AddDate(0, -1, 0).Format("2006-01")
			}
			items, err := h.Store.StallCosts(r.Context(), month, r.URL.Query().Get("q"))
			if err != nil {
				adminWriteError(w, err)
				return
			}
			httpx.JSON(w, http.StatusOK, map[string]any{"month": month, "items": items})
		case http.MethodPost, http.MethodPut:
			id, ok := adminID(w, r, r.Method == http.MethodPut)
			if !ok {
				return
			}
			var input request.StallCostInput
			if !decodeJSON(w, r, &input) {
				return
			}
			if err := input.Validate(); err != nil {
				httpx.Error(w, http.StatusBadRequest, err.Error())
				return
			}
			actor, ok := adminActor(w, r, tokens)
			if !ok {
				return
			}
			saved, err := h.Store.SaveStallCost(r.Context(), id, actor, input)
			if err != nil {
				adminWriteError(w, err)
				return
			}
			httpx.JSON(w, createdStatus(r.Method), map[string]any{"id": saved})
		case http.MethodDelete:
			id, ok := adminID(w, r, true)
			if !ok {
				return
			}
			actor, ok := adminActor(w, r, tokens)
			if !ok {
				return
			}
			if err := h.Store.DeleteStallCost(r.Context(), id, actor); err != nil {
				adminWriteError(w, err)
				return
			}
			httpx.JSON(w, http.StatusOK, map[string]any{"id": id})
		default:
			httpx.MethodNotAllowed(w, http.MethodGet, http.MethodPost, http.MethodPut, http.MethodDelete)
		}
	}
}

// StallCostLedgerAPI 将单条自营档口成本登记为台账支出。
func (h *Handler) StallCostLedgerAPI(tokens *token.Manager) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			httpx.MethodNotAllowed(w, http.MethodPost)
			return
		}
		actor, ok := adminActor(w, r, tokens)
		if !ok {
			return
		}
		var input request.StallCostLedgerInput
		if !decodeJSON(w, r, &input) {
			return
		}
		if err := input.Validate(); err != nil {
			httpx.Error(w, http.StatusBadRequest, err.Error())
			return
		}
		if err := h.Store.PostStallCostLedger(r.Context(), actor, input); err != nil {
			adminWriteError(w, err)
			return
		}
		httpx.JSON(w, http.StatusOK, map[string]any{"id": input.CostID})
	}
}

// StallMonthlyExpenseAPI 保存月度核算使用的水电及工资金额。
func (h *Handler) StallMonthlyExpenseAPI(tokens *token.Manager) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost && r.Method != http.MethodPut {
			httpx.MethodNotAllowed(w, http.MethodPost, http.MethodPut)
			return
		}
		actor, ok := adminActor(w, r, tokens)
		if !ok {
			return
		}
		var input request.StallMonthlyExpenseInput
		if !decodeJSON(w, r, &input) {
			return
		}
		if err := input.Validate(); err != nil {
			httpx.Error(w, http.StatusBadRequest, err.Error())
			return
		}
		if err := h.Store.SaveMonthlyOperatingExpense(r.Context(), actor, input); err != nil {
			adminWriteError(w, err)
			return
		}
		httpx.JSON(w, http.StatusOK, map[string]any{"month": input.Month})
	}
}

// StallMonthlyExpenseLedgerAPI 将月度核算经营支出一次性写入台账。
func (h *Handler) StallMonthlyExpenseLedgerAPI(tokens *token.Manager) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			httpx.MethodNotAllowed(w, http.MethodPost)
			return
		}
		actor, ok := adminActor(w, r, tokens)
		if !ok {
			return
		}
		var input request.StallMonthlyExpenseLedgerInput
		if !decodeJSON(w, r, &input) {
			return
		}
		if err := input.Validate(); err != nil {
			httpx.Error(w, http.StatusBadRequest, err.Error())
			return
		}
		if err := h.Store.PostMonthlyOperatingExpenseLedger(r.Context(), actor, input); err != nil {
			adminWriteError(w, err)
			return
		}
		httpx.JSON(w, http.StatusOK, map[string]any{"month": input.Month})
	}
}

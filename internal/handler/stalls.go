package handler

import (
	"fmt"
	"net/http"
	"strconv"
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
	ActiveMenu   string
	Month        string
	Keyword      string
	Stalls       []response.Stall
	Summary      response.StallSummary
	Settings     response.StallSettings
	Expenses     response.StallMonthlyExpense
	ProfitShares []response.ProfitShare
	// HasMonthlyProfitShareOverride 标记当前月份是否使用独立分成，而非公共规则。
	HasMonthlyProfitShareOverride bool
}

type StallCostsPageData struct {
	ActiveMenu     string
	Month          string
	Keyword        string
	SelfSummaries  []StallCostSummary
	OtherSummaries []StallCostSummary
	Stalls         []response.Stall
}

// StallCostSummary 是一个档口在当月的成本汇总，供结果页展示使用。
// 供货商拆分仍保留在供货商对账表，避免在档口成本页重复展示明细。
type StallCostSummary struct {
	Seq              int
	StallID          uint64
	StallName        string
	IngredientAmount string
	RiceOilAmount    string
	SpecialAmount    string
	TotalAmount      string
	EntryCount       int
}

// SupplierCostsPageData 是按月登记“供货商-档口”成本的对账页。
// 保存后的食材和米油金额与档口成本使用同一条记录，避免两边重复维护。
type SupplierCostsPageData struct {
	ActiveMenu string
	Month      string
	Stalls     []response.Stall
	Suppliers  []response.Supplier
	Columns    []SupplierCostMatrixColumn
	Rows       []SupplierCostMatrixRow
	EntryCount int
	GrandTotal string
}

// SupplierCostMatrixColumn 是供货商对账矩阵中的一个档口列及其月度汇总。
type SupplierCostMatrixColumn struct {
	StallID     uint64
	StallName   string
	TotalAmount string
}

// SupplierCostMatrixCell 是供货商和档口交叉的一格成本数据。
type SupplierCostMatrixCell struct {
	CostID           uint64
	StallID          uint64
	SupplierName     string
	IngredientAmount string
	RiceOilAmount    string
	TotalAmount      string
	HasEntry         bool
	LedgerPosted     bool
	ReadOnly         bool
	RiceOilOnly      bool
}

// SupplierCostMatrixRow 是一个供货商及其横向档口成本数据。
type SupplierCostMatrixRow struct {
	SupplierName  string
	Cells         []SupplierCostMatrixCell
	TotalAmount   string
	ReferenceOnly bool
}

type SuppliersPageData struct {
	ActiveMenu string
	Suppliers  []response.Supplier
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
	shares, hasMonthlyProfitShareOverride, err := h.Store.ProfitSharesWithSource(r.Context(), month)
	if err != nil {
		fail(w, err, "当月分成参与方读取失败")
		return
	}
	mysql.ApplyProfitShares(&summary, shares)
	expenses.ConradAmount = summary.ConradCommission
	expenses.CardIncomeAmount = summary.TotalCardTurnover
	expenses.HuilaimiIncomeAmount = summary.TotalHuilaimi
	h.render(w, "stall_settlement.html", StallMonthlyPageData{ActiveMenu: "stall-monthly", Month: month, Keyword: keyword, Stalls: items, Summary: summary, Settings: settings, Expenses: expenses, ProfitShares: summary.ProfitShares, HasMonthlyProfitShareOverride: hasMonthlyProfitShareOverride})
}

// StallCostsPage 按档口展示成本汇总，供货商拆分仅在供货商对账页维护。
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
	selfSummaries, otherSummaries := summarizeStallCosts(stalls, items)
	h.render(w, "stall_costs.html", StallCostsPageData{
		ActiveMenu:     "stall-costs",
		Month:          month,
		Keyword:        keyword,
		SelfSummaries:  selfSummaries,
		OtherSummaries: otherSummaries,
		Stalls:         stalls,
	})
}

// summarizeStallCosts 将同一档口的供货商成本和特殊调整分别汇总，避免手动金额混入供货商对账。
func summarizeStallCosts(stalls []response.Stall, items []response.StallCostEntry) ([]StallCostSummary, []StallCostSummary) {
	type aggregate struct {
		id         uint64
		name       string
		ingredient float64
		riceOil    float64
		special    float64
		count      int
	}
	selfOrder, otherOrder := make([]string, 0), make([]string, 0)
	selfData, otherData := make(map[string]*aggregate), make(map[string]*aggregate)
	for _, item := range items {
		target, order := otherData, &otherOrder
		if item.SelfOperated {
			target, order = selfData, &selfOrder
		}
		key := strconv.FormatUint(item.StallID, 10)
		entry, exists := target[key]
		if !exists {
			entry = &aggregate{id: item.StallID, name: item.StallName}
			target[key] = entry
			*order = append(*order, key)
		}
		ingredient, _ := strconv.ParseFloat(item.IngredientAmount, 64)
		riceOil, _ := strconv.ParseFloat(item.RiceOilAmount, 64)
		if item.SupplierName == "" {
			entry.special += ingredient + riceOil
		} else {
			entry.ingredient += ingredient
			entry.riceOil += riceOil
			entry.count++
		}
	}
	// 即使当月还没有供货商对账，也展示档口，方便直接添加特殊调整金额。
	for _, stall := range stalls {
		target, order := otherData, &otherOrder
		if stall.AllocationMode == "profit_share" {
			target, order = selfData, &selfOrder
		}
		key := strconv.FormatUint(stall.ID, 10)
		if _, exists := target[key]; !exists {
			target[key] = &aggregate{id: stall.ID, name: stall.Name}
			*order = append(*order, key)
		}
	}
	build := func(order []string, data map[string]*aggregate) []StallCostSummary {
		result := make([]StallCostSummary, 0, len(order))
		for index, key := range order {
			entry := data[key]
			result = append(result, StallCostSummary{
				Seq:              index + 1,
				StallID:          entry.id,
				StallName:        entry.name,
				IngredientAmount: matrixMoney(entry.ingredient),
				RiceOilAmount:    matrixMoney(entry.riceOil),
				SpecialAmount:    matrixMoney(entry.special),
				TotalAmount:      matrixMoney(entry.ingredient + entry.riceOil + entry.special),
				EntryCount:       entry.count,
			})
		}
		return result
	}
	return build(selfOrder, selfData), build(otherOrder, otherData)
}

func (h *Handler) SupplierCostsPage(w http.ResponseWriter, r *http.Request) {
	if !getOnly(w, r) {
		return
	}
	month := r.URL.Query().Get("month")
	if _, err := time.Parse("2006-01", month); err != nil {
		month = time.Now().AddDate(0, -1, 0).Format("2006-01")
	}
	items, err := h.Store.StallCosts(r.Context(), month, "")
	if err != nil {
		fail(w, err, "供货商对账读取失败")
		return
	}
	stalls, err := h.Store.CostStalls(r.Context())
	if err != nil {
		fail(w, err, "档口选项读取失败")
		return
	}
	suppliers, err := h.Store.Suppliers(r.Context())
	if err != nil {
		fail(w, err, "供货商资料读取失败")
		return
	}
	columns, rows, entryCount, grandTotal := buildSupplierCostMatrix(stalls, suppliers, items)
	h.render(w, "supplier_costs.html", SupplierCostsPageData{ActiveMenu: "supplier-costs", Month: month, Stalls: stalls, Suppliers: suppliers, Columns: columns, Rows: rows, EntryCount: entryCount, GrandTotal: grandTotal})
}

// buildSupplierCostMatrix 将明细整理为“行是供货商、列是档口”的对账矩阵。
func buildSupplierCostMatrix(stalls []response.Stall, suppliers []response.Supplier, items []response.StallCostEntry) ([]SupplierCostMatrixColumn, []SupplierCostMatrixRow, int, string) {
	columns := make([]SupplierCostMatrixColumn, 0, len(stalls))
	stallIndex := make(map[uint64]int, len(stalls))
	columnTotals := make([]float64, len(stalls))
	for index, stall := range stalls {
		columns = append(columns, SupplierCostMatrixColumn{StallID: stall.ID, StallName: stall.Name})
		stallIndex[stall.ID] = index
	}

	matrixSuppliers := append([]response.Supplier(nil), suppliers...)
	supplierNames := make(map[string]bool, len(matrixSuppliers))
	for _, supplier := range matrixSuppliers {
		supplierNames[supplier.Name] = true
	}
	type aggregate struct {
		id, count    uint64
		ingredient   float64
		riceOil      float64
		ledgerPosted bool
	}
	values := make(map[string]*aggregate)
	entryCount := 0
	for _, item := range items {
		if item.SupplierName == "" {
			continue
		}
		column, found := stallIndex[item.StallID]
		if !found {
			continue
		}
		if !supplierNames[item.SupplierName] {
			matrixSuppliers = append(matrixSuppliers, response.Supplier{Name: item.SupplierName})
			supplierNames[item.SupplierName] = true
		}
		ingredient, _ := strconv.ParseFloat(item.IngredientAmount, 64)
		riceOil, _ := strconv.ParseFloat(item.RiceOilAmount, 64)
		key := item.SupplierName + "\x00" + strconv.FormatUint(item.StallID, 10)
		cell := values[key]
		if cell == nil {
			cell = &aggregate{id: item.ID}
			values[key] = cell
		}
		cell.count++
		cell.ingredient += ingredient
		cell.riceOil += riceOil
		cell.ledgerPosted = cell.ledgerPosted || item.LedgerPosted
		// 米油是自营档口统一采购后分发给其他档口的领用明细，仅展示，不重复累计采购成本。
		if item.SupplierName != "米油" {
			columnTotals[column] += ingredient + riceOil
		}
		entryCount++
	}

	rows := make([]SupplierCostMatrixRow, 0, len(matrixSuppliers))
	grandTotal := 0.0
	for _, supplier := range matrixSuppliers {
		row := SupplierCostMatrixRow{SupplierName: supplier.Name, ReferenceOnly: supplier.Name == "米油", Cells: make([]SupplierCostMatrixCell, 0, len(columns))}
		rowTotal := 0.0
		for _, column := range columns {
			cell := SupplierCostMatrixCell{StallID: column.StallID, SupplierName: supplier.Name, RiceOilOnly: supplier.Name == "米油"}
			if value := values[supplier.Name+"\x00"+strconv.FormatUint(column.StallID, 10)]; value != nil {
				cell.CostID = value.id
				cell.IngredientAmount = matrixMoney(value.ingredient)
				cell.RiceOilAmount = matrixMoney(value.riceOil)
				cell.TotalAmount = matrixMoney(value.ingredient + value.riceOil)
				cell.HasEntry = true
				cell.LedgerPosted = value.ledgerPosted
				// 一个交叉格存在多条旧明细时只展示合计，禁止覆盖，避免历史数据丢失。
				cell.ReadOnly = value.count > 1 || value.ledgerPosted
				if !row.ReferenceOnly {
					rowTotal += value.ingredient + value.riceOil
				}
			}
			row.Cells = append(row.Cells, cell)
		}
		row.TotalAmount = matrixMoney(rowTotal)
		if !row.ReferenceOnly {
			grandTotal += rowTotal
		}
		rows = append(rows, row)
	}
	for index := range columns {
		columns[index].TotalAmount = matrixMoney(columnTotals[index])
	}
	return columns, rows, entryCount, matrixMoney(grandTotal)
}

func matrixMoney(value float64) string { return fmt.Sprintf("%.2f", value) }

func (h *Handler) SuppliersPage(w http.ResponseWriter, r *http.Request) {
	if !getOnly(w, r) {
		return
	}
	items, err := h.Store.Suppliers(r.Context())
	if err != nil {
		fail(w, err, "供货商资料读取失败")
		return
	}
	h.render(w, "suppliers.html", SuppliersPageData{ActiveMenu: "supplier-costs", Suppliers: items})
}

func (h *Handler) StallsAPI(tokens *token.Manager) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		switch r.Method {
		case http.MethodGet:
			month := r.URL.Query().Get("month")
			if _, err := time.Parse("2006-01", month); err != nil {
				month = time.Now().AddDate(0, -1, 0).Format("2006-01")
			}
			if idText := r.URL.Query().Get("id"); idText != "" {
				id, err := strconv.ParseUint(idText, 10, 64)
				if err != nil || id == 0 {
					httpx.Error(w, http.StatusBadRequest, "档口ID不正确")
					return
				}
				item, err := h.Store.StallMonthlyRecord(r.Context(), id, month)
				if err != nil {
					adminWriteError(w, err)
					return
				}
				httpx.JSON(w, http.StatusOK, map[string]any{"stall": item, "month": month})
				return
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

// ProfitShareParticipantsAPI 维护公共分成规则或指定月份的专属覆盖。
func (h *Handler) ProfitShareParticipantsAPI(tokens *token.Manager) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPut && r.Method != http.MethodDelete {
			httpx.MethodNotAllowed(w, http.MethodPut, http.MethodDelete)
			return
		}
		actor, ok := adminActor(w, r, tokens)
		if !ok {
			return
		}
		if r.Method == http.MethodDelete {
			month := r.URL.Query().Get("month")
			if _, err := time.Parse("2006-01", month); err != nil {
				httpx.Error(w, http.StatusBadRequest, "月份格式不正确")
				return
			}
			if err := h.Store.DeleteMonthlyProfitShares(r.Context(), actor, month); err != nil {
				adminWriteError(w, err)
				return
			}
			httpx.JSON(w, http.StatusOK, map[string]any{"month": month, "scope": "global"})
			return
		}
		var input request.ProfitShareParticipantsInput
		if !decodeJSON(w, r, &input) {
			return
		}
		if err := input.Validate(); err != nil {
			httpx.Error(w, http.StatusBadRequest, err.Error())
			return
		}
		if err := h.Store.SaveProfitShares(r.Context(), actor, input); err != nil {
			adminWriteError(w, err)
			return
		}
		httpx.JSON(w, http.StatusOK, map[string]any{"month": input.Month, "scope": input.Scope})
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

// StallCostSpecialAdjustmentAPI 在成本汇总表中直接更新一个档口的特殊调整总额。
func (h *Handler) StallCostSpecialAdjustmentAPI(tokens *token.Manager) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPut {
			httpx.MethodNotAllowed(w, http.MethodPut)
			return
		}
		var input request.StallCostSpecialAdjustmentInput
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
		id, err := h.Store.SaveStallCostSpecialAdjustment(r.Context(), actor, input)
		if err != nil {
			adminWriteError(w, err)
			return
		}
		httpx.JSON(w, http.StatusOK, map[string]any{"id": id})
	}
}

// SupplierCostMatrixAPI 保存供货商对账矩阵中一个供货商和档口交叉格的费用。
func (h *Handler) SupplierCostMatrixAPI(tokens *token.Manager) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPut {
			httpx.MethodNotAllowed(w, http.MethodPut)
			return
		}
		actor, ok := adminActor(w, r, tokens)
		if !ok {
			return
		}
		var input request.SupplierMatrixCostInput
		if !decodeJSON(w, r, &input) {
			return
		}
		if err := input.Validate(); err != nil {
			httpx.Error(w, http.StatusBadRequest, err.Error())
			return
		}
		id, err := h.Store.SaveSupplierMatrixCost(r.Context(), actor, input)
		if err != nil {
			adminWriteError(w, err)
			return
		}
		httpx.JSON(w, http.StatusOK, map[string]any{"id": id})
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

// StallMonthlyExpenseLedgerRevokeAPI 撤销指定月份由档口月度核算生成的全部流水。
func (h *Handler) StallMonthlyExpenseLedgerRevokeAPI(tokens *token.Manager) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			httpx.MethodNotAllowed(w, http.MethodPost)
			return
		}
		actor, ok := adminActor(w, r, tokens)
		if !ok {
			return
		}
		var input request.StallMonthlyExpenseLedgerRevokeInput
		if !decodeJSON(w, r, &input) {
			return
		}
		if err := input.Validate(); err != nil {
			httpx.Error(w, http.StatusBadRequest, err.Error())
			return
		}
		if err := h.Store.RevokeMonthlyOperatingExpenseLedger(r.Context(), actor, input.Month); err != nil {
			adminWriteError(w, err)
			return
		}
		httpx.JSON(w, http.StatusOK, map[string]any{"month": input.Month})
	}
}

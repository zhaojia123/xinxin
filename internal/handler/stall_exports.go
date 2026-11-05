package handler

import (
	"bytes"
	"fmt"
	"net/http"
	"net/url"
	"strconv"
	"strings"

	"friends-records/api/response"
	"friends-records/internal/httpx"
	"friends-records/internal/models/mysql"
	"friends-records/internal/token"

	"github.com/xuri/excelize/v2"
)

// SupplierCostsExport 导出指定月份的供货商-档口横向对账表。
func (h *Handler) SupplierCostsExport(tokens *token.Manager) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if _, ok := adminActor(w, r, tokens); !ok {
			return
		}
		if r.Method != http.MethodGet {
			httpx.MethodNotAllowed(w, http.MethodGet)
			return
		}
		month := mysql.NormalizeMonth(r.URL.Query().Get("month"))
		items, err := h.Store.StallCosts(r.Context(), month, "")
		if err != nil {
			fail(w, err, "供货商对账导出失败")
			return
		}
		stalls, err := h.Store.CostStalls(r.Context())
		if err != nil {
			fail(w, err, "供货商对账导出失败")
			return
		}
		suppliers, err := h.Store.Suppliers(r.Context())
		if err != nil {
			fail(w, err, "供货商对账导出失败")
			return
		}
		columns, rows, _, grandTotal := buildSupplierCostMatrix(stalls, suppliers, items)
		canteenName := ""
		if settings, settingsErr := h.Store.StallSettings(r.Context(), month); settingsErr == nil {
			canteenName = settings.CanteenName
		}
		if err := writeSupplierCostsXLSX(w, "供货商档口对账_"+month+".xlsx", supplierCostsExportTitle(month, canteenName), columns, rows, grandTotal); err != nil {
			fail(w, err, "供货商对账导出失败")
		}
	}
}

// StallMonthlyExport 导出指定月份的档口营业额、抽成和成本核算表。
func (h *Handler) StallMonthlyExport(tokens *token.Manager) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if _, ok := adminActor(w, r, tokens); !ok {
			return
		}
		if r.Method != http.MethodGet {
			httpx.MethodNotAllowed(w, http.MethodGet)
			return
		}
		month := mysql.NormalizeMonth(r.URL.Query().Get("month"))
		stalls, _, err := h.Store.StallsWithTurnover(r.Context(), month, "")
		if err != nil {
			fail(w, err, "档口月度核算导出失败")
			return
		}
		if err := writeStallMonthlyXLSX(w, "档口月度核算_"+month+".xlsx", monthTitle(month)+"档口月度核算表", stalls); err != nil {
			fail(w, err, "档口月度核算导出失败")
		}
	}
}

// stallCommissionRate 以当前档口规则生成便于人工核对的抽成比例说明。
func stallCommissionRate(item response.Stall) string {
	if item.AllocationMode == "profit_share" {
		return "甲90% / 乙10%"
	}
	if item.AllocationMode == "none" {
		return "不抽成"
	}
	if item.ManualOwnCommissionSet {
		return "手动填写"
	}
	if nonZeroAmount(item.TotalAmountRate) {
		return item.TotalAmountRate + "%"
	}
	if nonZeroAmount(item.BaseAmountRate) && nonZeroAmount(item.BaseAmountFactor) {
		return item.BaseAmountRate + "% × " + item.BaseAmountFactor + "%"
	}
	if nonZeroAmount(item.GuaranteedRate) {
		return "保底 " + item.GuaranteedRate + "%"
	}
	if nonZeroAmount(item.MonthlyRent) || nonZeroAmount(item.FixedCommissionAmount) {
		return "固定金额"
	}
	return "全额"
}

func nonZeroAmount(value string) bool {
	amount, _ := strconv.ParseFloat(strings.TrimSpace(value), 64)
	return amount != 0
}

// stallMonthlyExportCommissionRate 返回导出表中按总营业额实际计算的抽成比例。
// 自营档口不在此表展示甲乙分成，统一显示 0。
func stallMonthlyExportCommissionRate(item response.Stall) string {
	if item.AllocationMode == "profit_share" {
		return "0"
	}
	if rate := exportAmount(item.TotalAmountRate); rate != 0 {
		return exportPercentage(rate)
	}
	if base, factor := exportAmount(item.BaseAmountRate), exportAmount(item.BaseAmountFactor); base != 0 && factor != 0 {
		return exportPercentage(base * factor / 100)
	}
	return "0"
}

func exportPercentage(value float64) string {
	text := strconv.FormatFloat(value, 'f', 2, 64)
	text = strings.TrimRight(strings.TrimRight(text, "0"), ".")
	return text + "%"
}

// stallMonthlyActualPayment 计算导出表的实际支付：自营档口按营业额扣食材和米油，其他档口保留未分配余额。
func stallMonthlyActualPayment(item response.Stall) float64 {
	if item.AllocationMode == "profit_share" {
		return exportAmount(item.TotalTurnover) - exportAmount(item.IngredientCostAmount) - exportAmount(item.RiceOilCostAmount)
	}
	return exportAmount(item.Remaining)
}

func stallMonthlyExportCommissionAmount(item response.Stall) float64 {
	if item.AllocationMode == "profit_share" {
		return 0
	}
	return exportAmount(item.OwnCommission)
}

func stallMonthlyExportCommissionCell(item response.Stall) exportCell {
	if item.AllocationMode == "profit_share" {
		return exportText("0")
	}
	return exportNumber(mysql.Money(stallMonthlyExportCommissionAmount(item)))
}

func exportAmount(value string) float64 {
	amount, _ := strconv.ParseFloat(strings.TrimPrefix(strings.TrimSpace(value), "¥"), 64)
	return amount
}

// exportMoneyText 按签字表样式输出金额文本，避免不同 Excel 版本自动附加货币符号。
func exportMoneyText(value string) string {
	amount := exportAmount(value)
	negative := amount < 0
	if negative {
		amount = -amount
	}
	text := strconv.FormatFloat(amount, 'f', 2, 64)
	parts := strings.SplitN(text, ".", 2)
	for index := len(parts[0]) - 3; index > 0; index -= 3 {
		parts[0] = parts[0][:index] + "," + parts[0][index:]
	}
	if negative {
		parts[0] = "-" + parts[0]
	}
	return strings.Join(parts, ".")
}

type exportCell struct {
	value   string
	number  bool
	integer bool
}

var excelNumberFormat = "#,##0.00"

func exportText(value string) exportCell { return exportCell{value: value} }

// exportInteger 用于序号、天数等整数列，避免 Excel 显示为 1.00、26.00。
func exportInteger(value string) exportCell {
	value = strings.TrimSpace(value)
	if _, err := strconv.ParseInt(value, 10, 64); err == nil {
		return exportCell{value: value, number: true, integer: true}
	}
	return exportText(value)
}

func exportNumber(value string) exportCell {
	value = strings.TrimPrefix(strings.TrimSpace(value), "¥")
	if _, err := strconv.ParseFloat(value, 64); err == nil {
		return exportCell{value: value, number: true}
	}
	return exportText(value)
}

// writeExcelXLSX 生成标准 XLSX，保证 Excel、WPS 都能直接打开，并统一白底黑字的打印版式。
func writeExcelXLSX(w http.ResponseWriter, filename, title, sheetName string, headers []string, records [][]exportCell, widths []int) error {
	file := excelize.NewFile()
	defer file.Close()
	defaultSheet := file.GetSheetName(0)
	if err := file.SetSheetName(defaultSheet, sheetName); err != nil {
		return err
	}
	lastColumn, err := excelize.ColumnNumberToName(len(headers))
	if err != nil {
		return err
	}
	borders := []excelize.Border{{Type: "left", Color: "000000", Style: 1}, {Type: "right", Color: "000000", Style: 1}, {Type: "top", Color: "000000", Style: 1}, {Type: "bottom", Color: "000000", Style: 1}}
	titleStyle, err := file.NewStyle(&excelize.Style{Font: &excelize.Font{Family: "微软雅黑", Size: 16, Bold: true, Color: "000000"}, Alignment: &excelize.Alignment{Horizontal: "center", Vertical: "center"}})
	if err != nil {
		return err
	}
	headerStyle, err := file.NewStyle(&excelize.Style{Font: &excelize.Font{Family: "微软雅黑", Size: 10, Bold: true, Color: "000000"}, Alignment: &excelize.Alignment{Horizontal: "center", Vertical: "center", WrapText: true}, Border: borders})
	if err != nil {
		return err
	}
	textStyle, err := file.NewStyle(&excelize.Style{Font: &excelize.Font{Family: "微软雅黑", Size: 10, Color: "000000"}, Alignment: &excelize.Alignment{Vertical: "center", WrapText: true}, Border: borders})
	if err != nil {
		return err
	}
	numberStyle, err := file.NewStyle(&excelize.Style{Font: &excelize.Font{Family: "微软雅黑", Size: 10, Color: "000000"}, Alignment: &excelize.Alignment{Horizontal: "right", Vertical: "center"}, CustomNumFmt: &excelNumberFormat, Border: borders})
	if err != nil {
		return err
	}
	integerStyle, err := file.NewStyle(&excelize.Style{Font: &excelize.Font{Family: "微软雅黑", Size: 10, Color: "000000"}, Alignment: &excelize.Alignment{Horizontal: "right", Vertical: "center"}, NumFmt: 1, Border: borders})
	if err != nil {
		return err
	}
	totalBorders := []excelize.Border{{Type: "left", Color: "000000", Style: 1}, {Type: "right", Color: "000000", Style: 1}, {Type: "top", Color: "000000", Style: 2}, {Type: "bottom", Color: "000000", Style: 1}}
	totalStyle, err := file.NewStyle(&excelize.Style{Font: &excelize.Font{Family: "微软雅黑", Size: 10, Bold: true, Color: "000000"}, Alignment: &excelize.Alignment{Vertical: "center", WrapText: true}, Border: totalBorders})
	if err != nil {
		return err
	}
	totalNumberStyle, err := file.NewStyle(&excelize.Style{Font: &excelize.Font{Family: "微软雅黑", Size: 10, Bold: true, Color: "000000"}, Alignment: &excelize.Alignment{Horizontal: "right", Vertical: "center"}, CustomNumFmt: &excelNumberFormat, Border: totalBorders})
	if err != nil {
		return err
	}
	totalIntegerStyle, err := file.NewStyle(&excelize.Style{Font: &excelize.Font{Family: "微软雅黑", Size: 10, Bold: true, Color: "000000"}, Alignment: &excelize.Alignment{Horizontal: "right", Vertical: "center"}, NumFmt: 1, Border: totalBorders})
	if err != nil {
		return err
	}
	if err := file.MergeCell(sheetName, "A1", lastColumn+"1"); err != nil {
		return err
	}
	if err := file.SetCellValue(sheetName, "A1", title); err != nil {
		return err
	}
	if err := file.SetCellStyle(sheetName, "A1", lastColumn+"1", titleStyle); err != nil {
		return err
	}
	if err := file.SetRowHeight(sheetName, 1, 32); err != nil {
		return err
	}
	if err := file.SetRowHeight(sheetName, 2, 12); err != nil {
		return err
	}
	for columnIndex, header := range headers {
		column, err := excelize.ColumnNumberToName(columnIndex + 1)
		if err != nil {
			return err
		}
		if err := file.SetCellValue(sheetName, fmt.Sprintf("%s3", column), header); err != nil {
			return err
		}
		if err := file.SetColWidth(sheetName, column, column, exportColumnWidth(widths, columnIndex)); err != nil {
			return err
		}
	}
	if err := file.SetCellStyle(sheetName, "A3", lastColumn+"3", headerStyle); err != nil {
		return err
	}
	if err := file.SetRowHeight(sheetName, 3, 34); err != nil {
		return err
	}
	for recordIndex, record := range records {
		row := recordIndex + 4
		for columnIndex, cell := range record {
			column, err := excelize.ColumnNumberToName(columnIndex + 1)
			if err != nil {
				return err
			}
			coordinate := fmt.Sprintf("%s%d", column, row)
			if cell.number {
				value, err := strconv.ParseFloat(cell.value, 64)
				if err != nil {
					return err
				}
				if err := file.SetCellValue(sheetName, coordinate, value); err != nil {
					return err
				}
			} else if err := file.SetCellValue(sheetName, coordinate, cell.value); err != nil {
				return err
			}
			style := textStyle
			if cell.number {
				style = numberStyle
				if cell.integer {
					style = integerStyle
				}
			}
			if recordIndex == len(records)-1 {
				style = totalStyle
				if cell.number {
					style = totalNumberStyle
					if cell.integer {
						style = totalIntegerStyle
					}
				}
			}
			if err := file.SetCellStyle(sheetName, coordinate, coordinate, style); err != nil {
				return err
			}
		}
		if err := file.SetRowHeight(sheetName, row, 28); err != nil {
			return err
		}
	}
	if err := file.SetPanes(sheetName, &excelize.Panes{Freeze: true, Split: false, YSplit: 3, TopLeftCell: "A4", ActivePane: "bottomLeft"}); err != nil {
		return err
	}
	var content bytes.Buffer
	if err := file.Write(&content); err != nil {
		return err
	}
	w.Header().Set("Content-Type", "application/vnd.openxmlformats-officedocument.spreadsheetml.sheet")
	w.Header().Set("Content-Disposition", `attachment; filename="export.xlsx"; filename*=UTF-8''`+url.QueryEscape(filename))
	_, err = w.Write(content.Bytes())
	return err
}

// writeStallMonthlyXLSX 生成档口月度核算表，并保留自营档口的实际支付结算口径。
func writeStallMonthlyXLSX(w http.ResponseWriter, filename, title string, stalls []response.Stall) error {
	file := excelize.NewFile()
	defer file.Close()
	sheetName := "档口月度核算"
	if err := file.SetSheetName(file.GetSheetName(0), sheetName); err != nil {
		return err
	}
	headers := []string{"序号", "档口名称", "负责人", "学校卡机营业额", "汇来米营业额", "合计总金额", "抽成比例", "抽成", "食材费用", "米油费用", "实际支付", "负责人签字", "备注"}
	widths := []float64{8, 18, 14, 16, 16, 16, 14, 14, 14, 14, 16, 18, 24}
	lastColumn, err := excelize.ColumnNumberToName(len(headers))
	if err != nil {
		return err
	}
	thinBorders := []excelize.Border{{Type: "left", Color: "000000", Style: 1}, {Type: "right", Color: "000000", Style: 1}, {Type: "top", Color: "000000", Style: 1}, {Type: "bottom", Color: "000000", Style: 1}}
	yellowFill := excelize.Fill{Type: "pattern", Color: []string{"FFC000"}, Pattern: 1}
	titleStyle, err := file.NewStyle(&excelize.Style{Font: &excelize.Font{Family: "微软雅黑", Size: 16, Bold: true, Color: "000000"}, Alignment: &excelize.Alignment{Horizontal: "center", Vertical: "center"}})
	if err != nil {
		return err
	}
	headerStyle, err := file.NewStyle(&excelize.Style{Font: &excelize.Font{Family: "微软雅黑", Size: 10, Bold: true, Color: "000000"}, Alignment: &excelize.Alignment{Horizontal: "center", Vertical: "center", WrapText: true}, Border: thinBorders})
	if err != nil {
		return err
	}
	yellowHeaderStyle, err := file.NewStyle(&excelize.Style{Font: &excelize.Font{Family: "微软雅黑", Size: 10, Bold: true, Color: "000000"}, Alignment: &excelize.Alignment{Horizontal: "center", Vertical: "center", WrapText: true}, Fill: yellowFill, Border: thinBorders})
	if err != nil {
		return err
	}
	textStyle, err := file.NewStyle(&excelize.Style{Font: &excelize.Font{Family: "微软雅黑", Size: 10, Color: "000000"}, Alignment: &excelize.Alignment{Horizontal: "center", Vertical: "center", WrapText: true}, Border: thinBorders})
	if err != nil {
		return err
	}
	boldTextStyle, err := file.NewStyle(&excelize.Style{Font: &excelize.Font{Family: "微软雅黑", Size: 10, Bold: true, Color: "000000"}, Alignment: &excelize.Alignment{Horizontal: "center", Vertical: "center", WrapText: true}, Border: thinBorders})
	if err != nil {
		return err
	}
	moneyStyle, err := file.NewStyle(&excelize.Style{Font: &excelize.Font{Family: "微软雅黑", Size: 10, Color: "000000"}, Alignment: &excelize.Alignment{Horizontal: "center", Vertical: "center"}, CustomNumFmt: &excelNumberFormat, Border: thinBorders})
	if err != nil {
		return err
	}
	integerStyle, err := file.NewStyle(&excelize.Style{Font: &excelize.Font{Family: "微软雅黑", Size: 10, Bold: true, Color: "000000"}, Alignment: &excelize.Alignment{Horizontal: "center", Vertical: "center"}, NumFmt: 1, Border: thinBorders})
	if err != nil {
		return err
	}
	yellowMoneyStyle, err := file.NewStyle(&excelize.Style{Font: &excelize.Font{Family: "微软雅黑", Size: 10, Color: "000000"}, Alignment: &excelize.Alignment{Horizontal: "center", Vertical: "center"}, CustomNumFmt: &excelNumberFormat, Fill: yellowFill, Border: thinBorders})
	if err != nil {
		return err
	}
	yellowBoldMoneyStyle, err := file.NewStyle(&excelize.Style{Font: &excelize.Font{Family: "微软雅黑", Size: 10, Bold: true, Color: "000000"}, Alignment: &excelize.Alignment{Horizontal: "center", Vertical: "center"}, CustomNumFmt: &excelNumberFormat, Fill: yellowFill, Border: thinBorders})
	if err != nil {
		return err
	}
	boldMoneyStyle, err := file.NewStyle(&excelize.Style{Font: &excelize.Font{Family: "微软雅黑", Size: 10, Bold: true, Color: "000000"}, Alignment: &excelize.Alignment{Horizontal: "center", Vertical: "center"}, CustomNumFmt: &excelNumberFormat, Border: thinBorders})
	if err != nil {
		return err
	}

	if err := file.MergeCell(sheetName, "A1", lastColumn+"1"); err != nil {
		return err
	}
	if err := file.SetCellValue(sheetName, "A1", title); err != nil {
		return err
	}
	if err := file.SetCellStyle(sheetName, "A1", lastColumn+"1", titleStyle); err != nil {
		return err
	}
	if err := file.SetRowHeight(sheetName, 1, 32); err != nil {
		return err
	}
	if err := file.SetRowHeight(sheetName, 2, 10); err != nil {
		return err
	}
	for index, header := range headers {
		column, err := excelize.ColumnNumberToName(index + 1)
		if err != nil {
			return err
		}
		if err := file.SetCellValue(sheetName, column+"3", header); err != nil {
			return err
		}
		if err := file.SetColWidth(sheetName, column, column, widths[index]); err != nil {
			return err
		}
		style := headerStyle
		if index == 5 || index == 7 || index == 8 {
			style = yellowHeaderStyle
		}
		if err := file.SetCellStyle(sheetName, column+"3", column+"3", style); err != nil {
			return err
		}
	}
	if err := file.SetRowHeight(sheetName, 3, 34); err != nil {
		return err
	}

	mergedPaymentStart, mergedPaymentEnd, mergedPayment := monthlyCombinedPaymentRows(stalls)
	totals := stallMonthlyExportTotals(stalls)
	for index, item := range stalls {
		excelRow := index + 4
		values := []exportCell{
			exportInteger(strconv.Itoa(item.Seq)), exportText(item.Name), exportText(item.ResponsibleName), exportNumber(item.SchoolCardAmount), exportNumber(item.HuilaimiAmount), exportNumber(item.TotalTurnover), exportText(stallMonthlyExportCommissionRate(item)), stallMonthlyExportCommissionCell(item), exportNumber(item.IngredientCostAmount), exportNumber(item.RiceOilCostAmount), exportNumber(mysql.Money(stallMonthlyActualPayment(item))), exportText(""), exportText(item.Remark),
		}
		for columnIndex, cell := range values {
			if columnIndex == 10 && index == mergedPaymentEnd {
				continue
			}
			column, err := excelize.ColumnNumberToName(columnIndex + 1)
			if err != nil {
				return err
			}
			coordinate := fmt.Sprintf("%s%d", column, excelRow)
			if cell.number && !cell.integer {
				if err := file.SetCellStr(sheetName, coordinate, exportMoneyText(cell.value)); err != nil {
					return err
				}
			} else if cell.number {
				amount, err := strconv.ParseFloat(cell.value, 64)
				if err != nil {
					return err
				}
				if err := file.SetCellValue(sheetName, coordinate, amount); err != nil {
					return err
				}
			} else if err := file.SetCellValue(sheetName, coordinate, cell.value); err != nil {
				return err
			}
			style := textStyle
			if cell.number {
				style = moneyStyle
				if cell.integer {
					style = integerStyle
				}
			}
			if columnIndex == 1 {
				style = boldTextStyle
			}
			if columnIndex == 5 || columnIndex == 7 || columnIndex == 8 {
				style = yellowMoneyStyle
			}
			if err := file.SetCellStyle(sheetName, coordinate, coordinate, style); err != nil {
				return err
			}
		}
		if err := file.SetRowHeight(sheetName, excelRow, 28); err != nil {
			return err
		}
	}
	if mergedPaymentStart >= 0 && mergedPaymentEnd >= mergedPaymentStart {
		start := fmt.Sprintf("K%d", mergedPaymentStart+4)
		end := fmt.Sprintf("K%d", mergedPaymentEnd+4)
		if err := file.MergeCell(sheetName, start, end); err != nil {
			return err
		}
		if err := file.SetCellStr(sheetName, start, exportMoneyText(mysql.Money(mergedPayment))); err != nil {
			return err
		}
		if err := file.SetCellStyle(sheetName, start, end, moneyStyle); err != nil {
			return err
		}
	}

	totalRow := len(stalls) + 4
	totalValues := []exportCell{
		exportText("合计"), exportText(""), exportText(""), exportNumber(mysql.Money(totals.schoolCard)), exportNumber(mysql.Money(totals.huilaimi)), exportNumber(mysql.Money(totals.turnover)), exportText(""), exportNumber(mysql.Money(totals.commission)), exportNumber(mysql.Money(totals.ingredient)), exportNumber(mysql.Money(totals.riceOil)), exportNumber(mysql.Money(totals.actualPayment)), exportText(""), exportText(""),
	}
	for index, cell := range totalValues {
		column, err := excelize.ColumnNumberToName(index + 1)
		if err != nil {
			return err
		}
		coordinate := fmt.Sprintf("%s%d", column, totalRow)
		if cell.number && !cell.integer {
			if err := file.SetCellStr(sheetName, coordinate, exportMoneyText(cell.value)); err != nil {
				return err
			}
		} else if cell.number {
			amount, err := strconv.ParseFloat(cell.value, 64)
			if err != nil {
				return err
			}
			if err := file.SetCellValue(sheetName, coordinate, amount); err != nil {
				return err
			}
		} else if err := file.SetCellValue(sheetName, coordinate, cell.value); err != nil {
			return err
		}
		style := boldTextStyle
		if cell.number {
			style = boldMoneyStyle
		}
		if index == 5 || index == 7 || index == 8 {
			style = yellowBoldMoneyStyle
		}
		if err := file.SetCellStyle(sheetName, coordinate, coordinate, style); err != nil {
			return err
		}
	}
	if err := file.SetRowHeight(sheetName, totalRow, 30); err != nil {
		return err
	}
	if err := file.SetPanes(sheetName, &excelize.Panes{Freeze: true, Split: false, YSplit: 3, TopLeftCell: "A4", ActivePane: "bottomLeft"}); err != nil {
		return err
	}
	var content bytes.Buffer
	if err := file.Write(&content); err != nil {
		return err
	}
	w.Header().Set("Content-Type", "application/vnd.openxmlformats-officedocument.spreadsheetml.sheet")
	w.Header().Set("Content-Disposition", `attachment; filename="export.xlsx"; filename*=UTF-8''`+url.QueryEscape(filename))
	_, err = w.Write(content.Bytes())
	return err
}

type stallMonthlyTotals struct {
	schoolCard, huilaimi, turnover, commission, ingredient, riceOil, actualPayment float64
}

func stallMonthlyExportTotals(stalls []response.Stall) stallMonthlyTotals {
	var totals stallMonthlyTotals
	for _, item := range stalls {
		totals.schoolCard += exportAmount(item.SchoolCardAmount)
		totals.huilaimi += exportAmount(item.HuilaimiAmount)
		totals.turnover += exportAmount(item.TotalTurnover)
		totals.commission += stallMonthlyExportCommissionAmount(item)
		totals.ingredient += exportAmount(item.IngredientCostAmount)
		totals.riceOil += exportAmount(item.RiceOilCostAmount)
		totals.actualPayment += stallMonthlyActualPayment(item)
	}
	return totals
}

// monthlyCombinedPaymentRows 将大众快餐和自选餐的实际支付合并到大众快餐所在单元格。
func monthlyCombinedPaymentRows(stalls []response.Stall) (int, int, float64) {
	start, end := -1, -1
	for index, item := range stalls {
		switch item.Name {
		case "大众快餐":
			start = index
		case "自选餐":
			end = index
		}
	}
	if start < 0 || end != start+1 {
		return -1, -1, 0
	}
	return start, end, stallMonthlyActualPayment(stalls[start]) + stallMonthlyActualPayment(stalls[end])
}

func exportColumnWidth(widths []int, index int) float64 {
	if index >= len(widths) {
		return 16
	}
	width := float64(widths[index]) / 6
	if width < 12 {
		return 12
	}
	if width > 36 {
		return 36
	}
	return width
}

// supplierCostsExportTitle 使用食堂名称和月份生成供货商核对签字表标题。
func supplierCostsExportTitle(month, canteenName string) string {
	name := strings.TrimSpace(canteenName)
	if name == "" {
		name = "档口"
	}
	return name + "档口--" + monthTitle(month) + "供货商核对签字表"
}

// writeSupplierCostsXLSX 按月度供货商核对签字表的固定版式导出。
// 米油行是领用查看数据，不写入供货商金额及档口食材合计。
func writeSupplierCostsXLSX(w http.ResponseWriter, filename, title string, columns []SupplierCostMatrixColumn, rows []SupplierCostMatrixRow, grandTotal string) error {
	file := excelize.NewFile()
	defer file.Close()
	sheetName := "供货商核对签字表"
	if err := file.SetSheetName(file.GetSheetName(0), sheetName); err != nil {
		return err
	}
	lastColumn, err := excelize.ColumnNumberToName(len(columns) + 3)
	if err != nil {
		return err
	}
	thinBorders := []excelize.Border{{Type: "left", Color: "000000", Style: 1}, {Type: "right", Color: "000000", Style: 1}, {Type: "top", Color: "000000", Style: 1}, {Type: "bottom", Color: "000000", Style: 1}}
	greenFill := excelize.Fill{Type: "pattern", Color: []string{"C6E0B4"}, Pattern: 1}
	yellowFill := excelize.Fill{Type: "pattern", Color: []string{"FFC000"}, Pattern: 1}
	grayFill := excelize.Fill{Type: "pattern", Color: []string{"E7E6E6"}, Pattern: 1}
	titleStyle, err := file.NewStyle(&excelize.Style{Font: &excelize.Font{Family: "微软雅黑", Size: 18, Bold: true, Color: "000000"}, Alignment: &excelize.Alignment{Horizontal: "center", Vertical: "center"}})
	if err != nil {
		return err
	}
	headerStyle, err := file.NewStyle(&excelize.Style{Font: &excelize.Font{Family: "微软雅黑", Size: 10, Bold: true, Color: "000000"}, Alignment: &excelize.Alignment{Horizontal: "center", Vertical: "center", WrapText: true}, Border: thinBorders})
	if err != nil {
		return err
	}
	textStyle, err := file.NewStyle(&excelize.Style{Font: &excelize.Font{Family: "微软雅黑", Size: 10, Color: "000000"}, Alignment: &excelize.Alignment{Horizontal: "center", Vertical: "center", WrapText: true}, Border: thinBorders})
	if err != nil {
		return err
	}
	moneyStyle, err := file.NewStyle(&excelize.Style{Font: &excelize.Font{Family: "微软雅黑", Size: 10, Color: "000000"}, Alignment: &excelize.Alignment{Horizontal: "right", Vertical: "center"}, CustomNumFmt: &excelNumberFormat, Border: thinBorders})
	if err != nil {
		return err
	}
	greenTextStyle, err := file.NewStyle(&excelize.Style{Font: &excelize.Font{Family: "微软雅黑", Size: 10, Color: "000000"}, Alignment: &excelize.Alignment{Horizontal: "center", Vertical: "center", WrapText: true}, Fill: greenFill, Border: thinBorders})
	if err != nil {
		return err
	}
	greenBoldTextStyle, err := file.NewStyle(&excelize.Style{Font: &excelize.Font{Family: "微软雅黑", Size: 10, Bold: true, Color: "000000"}, Alignment: &excelize.Alignment{Horizontal: "center", Vertical: "center", WrapText: true}, Fill: greenFill, Border: thinBorders})
	if err != nil {
		return err
	}
	greenMoneyStyle, err := file.NewStyle(&excelize.Style{Font: &excelize.Font{Family: "微软雅黑", Size: 10, Color: "000000"}, Alignment: &excelize.Alignment{Horizontal: "right", Vertical: "center"}, CustomNumFmt: &excelNumberFormat, Fill: greenFill, Border: thinBorders})
	if err != nil {
		return err
	}
	grayMoneyStyle, err := file.NewStyle(&excelize.Style{Font: &excelize.Font{Family: "微软雅黑", Size: 10, Color: "000000"}, Alignment: &excelize.Alignment{Horizontal: "right", Vertical: "center"}, CustomNumFmt: &excelNumberFormat, Fill: grayFill, Border: thinBorders})
	if err != nil {
		return err
	}
	yellowTextStyle, err := file.NewStyle(&excelize.Style{Font: &excelize.Font{Family: "微软雅黑", Size: 10, Bold: true, Color: "000000"}, Alignment: &excelize.Alignment{Horizontal: "center", Vertical: "center", WrapText: true}, Fill: yellowFill, Border: thinBorders})
	if err != nil {
		return err
	}
	yellowMoneyStyle, err := file.NewStyle(&excelize.Style{Font: &excelize.Font{Family: "微软雅黑", Size: 10, Bold: true, Color: "000000"}, Alignment: &excelize.Alignment{Horizontal: "right", Vertical: "center"}, CustomNumFmt: &excelNumberFormat, Fill: yellowFill, Border: thinBorders})
	if err != nil {
		return err
	}

	if err := file.MergeCell(sheetName, "A1", lastColumn+"1"); err != nil {
		return err
	}
	if err := file.SetCellValue(sheetName, "A1", title); err != nil {
		return err
	}
	if err := file.SetCellStyle(sheetName, "A1", lastColumn+"1", titleStyle); err != nil {
		return err
	}
	if err := file.SetRowHeight(sheetName, 1, 42); err != nil {
		return err
	}
	if err := file.SetRowHeight(sheetName, 2, 10); err != nil {
		return err
	}

	headers := append([]string{"序号", "供货商"}, supplierColumnNames(columns)...)
	headers = append(headers, "供货商金额合计")
	for index, header := range headers {
		column, err := excelize.ColumnNumberToName(index + 1)
		if err != nil {
			return err
		}
		if err := file.SetCellValue(sheetName, column+"3", header); err != nil {
			return err
		}
		width := 15.0
		switch index {
		case 0:
			width = 8
		case 1:
			width = 18
		case len(headers) - 1:
			width = 18
		}
		if err := file.SetColWidth(sheetName, column, column, width); err != nil {
			return err
		}
	}
	if err := file.SetCellStyle(sheetName, "A3", lastColumn+"3", headerStyle); err != nil {
		return err
	}
	if err := file.SetCellStyle(sheetName, "B3", "B3", greenBoldTextStyle); err != nil {
		return err
	}
	if err := file.SetCellStyle(sheetName, lastColumn+"3", lastColumn+"3", greenBoldTextStyle); err != nil {
		return err
	}
	if err := file.SetRowHeight(sheetName, 3, 38); err != nil {
		return err
	}

	for index, row := range rows {
		excelRow := index + 4
		if err := file.SetCellValue(sheetName, fmt.Sprintf("A%d", excelRow), index+1); err != nil {
			return err
		}
		if err := file.SetCellStyle(sheetName, fmt.Sprintf("A%d", excelRow), fmt.Sprintf("A%d", excelRow), textStyle); err != nil {
			return err
		}
		if err := file.SetCellValue(sheetName, fmt.Sprintf("B%d", excelRow), row.SupplierName); err != nil {
			return err
		}
		if err := file.SetCellStyle(sheetName, fmt.Sprintf("B%d", excelRow), fmt.Sprintf("B%d", excelRow), greenBoldTextStyle); err != nil {
			return err
		}
		for columnIndex, cell := range row.Cells {
			column, err := excelize.ColumnNumberToName(columnIndex + 3)
			if err != nil {
				return err
			}
			coordinate := fmt.Sprintf("%s%d", column, excelRow)
			style := moneyStyle
			if row.ReferenceOnly {
				style = grayMoneyStyle
			}
			if strings.TrimSpace(cell.TotalAmount) != "" {
				amount, err := strconv.ParseFloat(cell.TotalAmount, 64)
				if err != nil {
					return err
				}
				if err := file.SetCellValue(sheetName, coordinate, amount); err != nil {
					return err
				}
			}
			if err := file.SetCellStyle(sheetName, coordinate, coordinate, style); err != nil {
				return err
			}
		}
		totalCoordinate := fmt.Sprintf("%s%d", lastColumn, excelRow)
		if row.ReferenceOnly {
			if err := file.SetCellValue(sheetName, totalCoordinate, "仅供查看"); err != nil {
				return err
			}
			if err := file.SetCellStyle(sheetName, totalCoordinate, totalCoordinate, greenTextStyle); err != nil {
				return err
			}
		} else {
			amount, err := strconv.ParseFloat(row.TotalAmount, 64)
			if err != nil {
				return err
			}
			if err := file.SetCellValue(sheetName, totalCoordinate, amount); err != nil {
				return err
			}
			if err := file.SetCellStyle(sheetName, totalCoordinate, totalCoordinate, greenMoneyStyle); err != nil {
				return err
			}
		}
		if err := file.SetRowHeight(sheetName, excelRow, 38); err != nil {
			return err
		}
	}

	totalRow := len(rows) + 4
	if err := file.MergeCell(sheetName, fmt.Sprintf("A%d", totalRow), fmt.Sprintf("B%d", totalRow)); err != nil {
		return err
	}
	if err := file.SetCellValue(sheetName, fmt.Sprintf("A%d", totalRow), "档口食材合计"); err != nil {
		return err
	}
	if err := file.SetCellStyle(sheetName, fmt.Sprintf("A%d", totalRow), fmt.Sprintf("B%d", totalRow), yellowTextStyle); err != nil {
		return err
	}
	for index, column := range columns {
		coordinateColumn, err := excelize.ColumnNumberToName(index + 3)
		if err != nil {
			return err
		}
		coordinate := fmt.Sprintf("%s%d", coordinateColumn, totalRow)
		amount, err := strconv.ParseFloat(column.TotalAmount, 64)
		if err != nil {
			return err
		}
		if err := file.SetCellValue(sheetName, coordinate, amount); err != nil {
			return err
		}
		if err := file.SetCellStyle(sheetName, coordinate, coordinate, yellowMoneyStyle); err != nil {
			return err
		}
	}
	grandAmount, err := strconv.ParseFloat(grandTotal, 64)
	if err != nil {
		return err
	}
	if err := file.SetCellValue(sheetName, fmt.Sprintf("%s%d", lastColumn, totalRow), grandAmount); err != nil {
		return err
	}
	if err := file.SetCellStyle(sheetName, fmt.Sprintf("%s%d", lastColumn, totalRow), fmt.Sprintf("%s%d", lastColumn, totalRow), yellowMoneyStyle); err != nil {
		return err
	}
	if err := file.SetRowHeight(sheetName, totalRow, 34); err != nil {
		return err
	}

	confirmRow := totalRow + 1
	if err := file.MergeCell(sheetName, fmt.Sprintf("A%d", confirmRow), fmt.Sprintf("B%d", confirmRow)); err != nil {
		return err
	}
	if err := file.SetCellValue(sheetName, fmt.Sprintf("A%d", confirmRow), "档口签字确认："); err != nil {
		return err
	}
	if err := file.SetCellStyle(sheetName, fmt.Sprintf("A%d", confirmRow), fmt.Sprintf("A%d", confirmRow), textStyle); err != nil {
		return err
	}
	if err := file.SetCellStyle(sheetName, fmt.Sprintf("B%d", confirmRow), lastColumn+strconv.Itoa(confirmRow), textStyle); err != nil {
		return err
	}
	if err := file.SetRowHeight(sheetName, confirmRow, 36); err != nil {
		return err
	}

	signatureRow := confirmRow + 2
	managerEnd := "D"
	if len(columns)+3 < 4 {
		managerEnd = lastColumn
	}
	if err := file.MergeCell(sheetName, fmt.Sprintf("A%d", signatureRow), managerEnd+strconv.Itoa(signatureRow)); err != nil {
		return err
	}
	if err := file.SetCellValue(sheetName, fmt.Sprintf("A%d", signatureRow), "经理签字：________________"); err != nil {
		return err
	}
	if err := file.SetCellStyle(sheetName, fmt.Sprintf("A%d", signatureRow), managerEnd+strconv.Itoa(signatureRow), textStyle); err != nil {
		return err
	}
	if managerEnd != lastColumn {
		if err := file.MergeCell(sheetName, "E"+strconv.Itoa(signatureRow), lastColumn+strconv.Itoa(signatureRow)); err != nil {
			return err
		}
		if err := file.SetCellValue(sheetName, "E"+strconv.Itoa(signatureRow), "总负责人签字：________________"); err != nil {
			return err
		}
		if err := file.SetCellStyle(sheetName, "E"+strconv.Itoa(signatureRow), lastColumn+strconv.Itoa(signatureRow), textStyle); err != nil {
			return err
		}
	}
	if err := file.SetRowHeight(sheetName, signatureRow, 34); err != nil {
		return err
	}
	if err := file.SetPanes(sheetName, &excelize.Panes{Freeze: true, Split: false, YSplit: 3, TopLeftCell: "A4", ActivePane: "bottomLeft"}); err != nil {
		return err
	}
	var content bytes.Buffer
	if err := file.Write(&content); err != nil {
		return err
	}
	w.Header().Set("Content-Type", "application/vnd.openxmlformats-officedocument.spreadsheetml.sheet")
	w.Header().Set("Content-Disposition", `attachment; filename="export.xlsx"; filename*=UTF-8''`+url.QueryEscape(filename))
	_, err = w.Write(content.Bytes())
	return err
}

func supplierColumnNames(columns []SupplierCostMatrixColumn) []string {
	names := make([]string, 0, len(columns))
	for _, column := range columns {
		names = append(names, column.StallName)
	}
	return names
}

func monthTitle(month string) string {
	parts := strings.Split(month, "-")
	if len(parts) != 2 {
		return month
	}
	return parts[0] + "年" + parts[1] + "月"
}

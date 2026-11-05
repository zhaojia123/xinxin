package handler

import (
	"bytes"
	"net/http/httptest"
	"testing"

	"friends-records/api/response"

	"github.com/xuri/excelize/v2"
)

func TestPayrollActualDaysHonorsRestDayAllowance(t *testing.T) {
	tests := []struct {
		name string
		item response.PayrollRecord
		want string
	}{
		{name: "公休在额度内满勤", item: response.PayrollRecord{RequiredAttendanceDays: 27, PublicRestDays: "3.00"}, want: "27"},
		{name: "超额公休才扣减", item: response.PayrollRecord{RequiredAttendanceDays: 27, PublicRestDays: "4.00"}, want: "26"},
		{name: "请假正常扣减", item: response.PayrollRecord{RequiredAttendanceDays: 27, PublicRestDays: "3.00", LeaveDays: "1.00"}, want: "26"},
		{name: "月中入职按实际在岗天数", item: response.PayrollRecord{Month: "2026-10", JoinedOn: "2026-10-08", RequiredAttendanceDays: 27, PublicRestDays: "1.00"}, want: "23"},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			if got := payrollActualDays(test.item); got != test.want {
				t.Fatalf("实际出勤天数 = %s，期望 %s", got, test.want)
			}
		})
	}
}

func TestExportInteger(t *testing.T) {
	cell := exportInteger("26")
	if !cell.number || !cell.integer || cell.value != "26" {
		t.Fatalf("整数导出单元格不正确：%+v", cell)
	}
}

func TestWriteExcelXLSXProducesStandardWorkbook(t *testing.T) {
	recorder := httptest.NewRecorder()
	err := writeExcelXLSX(recorder, "测试导出.xlsx", "2026年10月测试表", "测试表", []string{"序号", "金额"}, [][]exportCell{{exportInteger("1"), exportNumber("12.50")}, {exportText("合计"), exportNumber("12.50")}}, []int{48, 100})
	if err != nil {
		t.Fatalf("生成 XLSX 失败：%v", err)
	}
	if got := recorder.Header().Get("Content-Type"); got != "application/vnd.openxmlformats-officedocument.spreadsheetml.sheet" {
		t.Fatalf("导出类型错误：%s", got)
	}
	if !bytes.HasPrefix(recorder.Body.Bytes(), []byte("PK")) {
		t.Fatal("导出内容不是标准 XLSX 压缩包")
	}
	book, err := excelize.OpenReader(bytes.NewReader(recorder.Body.Bytes()))
	if err != nil {
		t.Fatalf("Excel 无法打开导出文件：%v", err)
	}
	defer book.Close()
	if title, err := book.GetCellValue("测试表", "A1"); err != nil || title != "2026年10月测试表" {
		t.Fatalf("标题读取错误：title=%q err=%v", title, err)
	}
	if value, err := book.GetCellValue("测试表", "B4"); err != nil || value != "12.50" {
		t.Fatalf("数值读取错误：value=%q err=%v", value, err)
	}
}

func TestWriteSupplierCostsXLSXProducesSignatureLayout(t *testing.T) {
	recorder := httptest.NewRecorder()
	columns := []SupplierCostMatrixColumn{{StallName: "档口A", TotalAmount: "10.00"}, {StallName: "档口B", TotalAmount: "0.00"}}
	rows := []SupplierCostMatrixRow{
		{SupplierName: "供应商甲", TotalAmount: "10.00", Cells: []SupplierCostMatrixCell{{TotalAmount: "10.00"}, {TotalAmount: ""}}},
		{SupplierName: "米油", ReferenceOnly: true, Cells: []SupplierCostMatrixCell{{TotalAmount: "2.00"}, {TotalAmount: "3.00"}}},
	}
	if err := writeSupplierCostsXLSX(recorder, "供货商档口对账.xlsx", "知行二餐厅档口--供货商核对签字表", columns, rows, "10.00"); err != nil {
		t.Fatalf("生成供货商签字表失败：%v", err)
	}
	book, err := excelize.OpenReader(bytes.NewReader(recorder.Body.Bytes()))
	if err != nil {
		t.Fatalf("Excel 无法打开供货商签字表：%v", err)
	}
	defer book.Close()
	sheet := "供货商核对签字表"
	checks := map[string]string{"A1": "知行二餐厅档口--供货商核对签字表", "A3": "序号", "B4": "供应商甲", "D4": "", "E5": "仅供查看", "A6": "档口食材合计", "A7": "档口签字确认：", "A9": "经理签字：________________", "E9": "总负责人签字：________________"}
	for cell, want := range checks {
		if got, err := book.GetCellValue(sheet, cell); err != nil || got != want {
			t.Fatalf("单元格 %s = %q，期望 %q，错误=%v", cell, got, want, err)
		}
	}
}

func TestWriteStallMonthlyXLSXUsesSettlementPaymentLayout(t *testing.T) {
	recorder := httptest.NewRecorder()
	stalls := []response.Stall{
		{Seq: 1, Name: "益禾堂", AllocationMode: "profit_share", TotalTurnover: "100.00", IngredientCostAmount: "30.00", RiceOilCostAmount: "5.00"},
		{Seq: 2, Name: "大众快餐", AllocationMode: "profit_share", TotalTurnover: "100.00", IngredientCostAmount: "20.00", RiceOilCostAmount: "10.00"},
		{Seq: 3, Name: "自选餐", AllocationMode: "profit_share", TotalTurnover: "50.00", IngredientCostAmount: "5.00", RiceOilCostAmount: "5.00"},
		{Seq: 4, Name: "其他档口", AllocationMode: "custom", TotalTurnover: "100.00", TotalAmountRate: "22.00", OwnCommission: "22.00", IngredientCostAmount: "10.00", Remaining: "68.00"},
	}
	if err := writeStallMonthlyXLSX(recorder, "档口月度核算.xlsx", "2026年09月档口月度核算表", stalls); err != nil {
		t.Fatalf("生成档口月度核算表失败：%v", err)
	}
	book, err := excelize.OpenReader(bytes.NewReader(recorder.Body.Bytes()))
	if err != nil {
		t.Fatalf("Excel 无法打开档口月度核算表：%v", err)
	}
	defer book.Close()
	sheet := "档口月度核算"
	checks := map[string]string{"F3": "合计总金额", "G4": "0", "H4": "0", "K4": "65.00", "K5": "110.00", "G7": "22%", "A8": "合计"}
	for cell, want := range checks {
		if got, err := book.GetCellValue(sheet, cell); err != nil || got != want {
			t.Fatalf("单元格 %s = %q，期望 %q，错误=%v", cell, got, want, err)
		}
	}
	if merged, err := book.GetMergeCells(sheet); err != nil || !hasMergeRange(merged, "K5:K6") {
		t.Fatalf("大众快餐和自选餐实际支付应合并，合并区域=%v，错误=%v", merged, err)
	}
}

func hasMergeRange(cells []excelize.MergeCell, want string) bool {
	for _, cell := range cells {
		if cell.GetStartAxis()+":"+cell.GetEndAxis() == want {
			return true
		}
	}
	return false
}

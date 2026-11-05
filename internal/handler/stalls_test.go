package handler

import (
	"testing"

	"friends-records/api/response"
)

func TestBuildSupplierCostMatrixCalculatesRowAndColumnTotals(t *testing.T) {
	stalls := []response.Stall{{ID: 1, Name: "档口A"}, {ID: 2, Name: "档口B"}}
	suppliers := []response.Supplier{{Name: "供应商甲"}, {Name: "供应商乙"}}
	items := []response.StallCostEntry{
		{ID: 1, StallID: 1, SupplierName: "供应商甲", IngredientAmount: "1.00", RiceOilAmount: "2.00"},
		{ID: 2, StallID: 2, SupplierName: "供应商甲", IngredientAmount: "4.00", RiceOilAmount: "0.00"},
		{ID: 3, StallID: 1, SupplierName: "供应商乙", IngredientAmount: "5.00", RiceOilAmount: "0.00"},
	}

	columns, rows, entryCount, grandTotal := buildSupplierCostMatrix(stalls, suppliers, items)

	if entryCount != 3 || grandTotal != "12.00" {
		t.Fatalf("明细数或总计错误，明细=%d，总计=%s", entryCount, grandTotal)
	}
	if columns[0].TotalAmount != "8.00" || columns[1].TotalAmount != "4.00" {
		t.Fatalf("档口列汇总错误，A=%s，B=%s", columns[0].TotalAmount, columns[1].TotalAmount)
	}
	if rows[0].TotalAmount != "7.00" || rows[1].TotalAmount != "5.00" {
		t.Fatalf("供货商行汇总错误，甲=%s，乙=%s", rows[0].TotalAmount, rows[1].TotalAmount)
	}
	if rows[0].Cells[0].TotalAmount != "3.00" || rows[0].Cells[0].RiceOilAmount != "2.00" {
		t.Fatalf("交叉格成本拆分错误，合计=%s，米油=%s", rows[0].Cells[0].TotalAmount, rows[0].Cells[0].RiceOilAmount)
	}
}

func TestBuildSupplierCostMatrixExcludesRiceOilReferenceFromTotals(t *testing.T) {
	stalls := []response.Stall{{ID: 1, Name: "档口A"}, {ID: 2, Name: "档口B"}}
	suppliers := []response.Supplier{{Name: "供应商甲"}, {Name: "米油"}}
	items := []response.StallCostEntry{
		{ID: 1, StallID: 1, SupplierName: "供应商甲", IngredientAmount: "10.00"},
		{ID: 2, StallID: 1, SupplierName: "米油", RiceOilAmount: "2.00"},
		{ID: 3, StallID: 2, SupplierName: "米油", RiceOilAmount: "3.00"},
	}

	columns, rows, _, grandTotal := buildSupplierCostMatrix(stalls, suppliers, items)

	if grandTotal != "10.00" || columns[0].TotalAmount != "10.00" || columns[1].TotalAmount != "0.00" {
		t.Fatalf("米油领用不应进入合计，档口A=%s，档口B=%s，总计=%s", columns[0].TotalAmount, columns[1].TotalAmount, grandTotal)
	}
	if !rows[1].ReferenceOnly || rows[1].Cells[0].TotalAmount != "2.00" || rows[1].Cells[1].TotalAmount != "3.00" {
		t.Fatalf("米油领用明细应保留展示：%+v", rows[1])
	}
}

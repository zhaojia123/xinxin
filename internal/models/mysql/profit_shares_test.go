package mysql

import (
	"testing"

	"friends-records/api/response"
)

func TestApplyProfitSharesSplitsLossByRate(t *testing.T) {
	summary := response.StallSummary{DistributableBalance: "-100.00", HasMonthlyLoss: true}
	shares := []response.ProfitShare{{Name: "张", Rate: "80.00"}, {Name: "林", Rate: "10.00"}, {Name: "白", Rate: "10.00"}}

	ApplyProfitShares(&summary, shares)

	if got := summary.ProfitShares[0].LossAmount; got != "80.00" {
		t.Fatalf("张应承担80元亏损，实际=%s", got)
	}
	if got := summary.ProfitShares[1].LossAmount; got != "10.00" {
		t.Fatalf("林应承担10元亏损，实际=%s", got)
	}
	if got := summary.ProfitShares[2].LossAmount; got != "10.00" {
		t.Fatalf("白应承担10元亏损，实际=%s", got)
	}
	for _, item := range summary.ProfitShares {
		if item.Amount != "0.00" {
			t.Fatalf("亏损月份不应产生分成，参与方=%s，分成=%s", item.Name, item.Amount)
		}
	}
}

func TestApplyProfitSharesSplitsProfitByRate(t *testing.T) {
	summary := response.StallSummary{DistributableBalance: "100.00"}
	shares := []response.ProfitShare{{Name: "张", Rate: "80.00"}, {Name: "林", Rate: "10.00"}, {Name: "白", Rate: "10.00"}}

	ApplyProfitShares(&summary, shares)

	if got := summary.ProfitShares[0].Amount; got != "80.00" {
		t.Fatalf("张应分成80元，实际=%s", got)
	}
	if got := summary.ProfitShares[0].LossAmount; got != "0.00" {
		t.Fatalf("盈利月份不应产生亏损分摊，实际=%s", got)
	}
}

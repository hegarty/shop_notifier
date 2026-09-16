package format

import (
	"strings"
	"testing"
)

func TestSalesChannelBreakdown_MatchesOriginalBriefExample(t *testing.T) {
	payload := AnalyticsResultPayload{
		TenantID: "devmoto",
		Job:      "sales.channel.breakdown",
	}
	payload.Period.Label = "Sep 11"
	payload.Sales = &SalesResult{
		Total:      "10254.00",
		Shop:       "6842.00",
		OrderCount: 45,
		AOV:        "227.87",
		Collective: CollectiveBreakdown{
			Total: "3412.00",
			Partners: []PartnerBreakdown{
				{Name: "E-Moto Co", Sales: "1282.00"},
				{Name: "Ride Electric", Sales: "934.00"},
				{Name: "XYZ Motors", Sales: "721.00"},
				{Name: "ABC Bikes", Sales: "475.00"},
			},
		},
	}

	// tenantDisplayName includes the period-kind word ("Daily") — this
	// package doesn't know the schedule's period kind, only its resolved
	// label ("Sep 11"), so assembling "DevMoto Daily" is the caller's job.
	// See the package doc comment.
	got, err := SalesChannelBreakdown("DevMoto Daily", payload)
	if err != nil {
		t.Fatalf("SalesChannelBreakdown: %v", err)
	}

	want := `DevMoto Daily — Sep 11

Total Sales: $10,254

Shop:
$6,842
66.7%

Collective:
$3,412
33.3%

Collective Breakdown:
E-Moto Co  $1,282
Ride Electric  $934
XYZ Motors  $721
ABC Bikes  $475

Orders: 45
AOV: $227.87`

	if got != want {
		t.Errorf("output mismatch:\n--- got ---\n%s\n--- want ---\n%s", got, want)
	}
}

func TestSalesChannelBreakdown_NoSalesData(t *testing.T) {
	if _, err := SalesChannelBreakdown("DevMoto", AnalyticsResultPayload{}); err == nil {
		t.Fatal("expected error when Sales is nil")
	}
}

func TestFormatDollars(t *testing.T) {
	cases := []struct {
		cents int64
		want  string
	}{
		{1025400, "$10,254"},
		{22787, "$227.87"},
		{500, "$5"},
		{0, "$0"},
		{-1250, "-$12.50"},
		{99, "$0.99"},
	}
	for _, c := range cases {
		if got := formatDollars(c.cents); got != c.want {
			t.Errorf("formatDollars(%d) = %q, want %q", c.cents, got, c.want)
		}
	}
}

func TestSalesChannelBreakdown_NoPartners_OmitsSection(t *testing.T) {
	payload := AnalyticsResultPayload{}
	payload.Period.Label = "Sep 11"
	payload.Sales = &SalesResult{Total: "100.00", Shop: "100.00", OrderCount: 1, AOV: "100.00"}

	got, err := SalesChannelBreakdown("DevMoto Daily", payload)
	if err != nil {
		t.Fatalf("SalesChannelBreakdown: %v", err)
	}
	if strings.Contains(got, "Collective Breakdown") {
		t.Error("expected no 'Collective Breakdown' section when there are no partners")
	}
}

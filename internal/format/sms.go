// Package format renders analytics results as human-readable text —
// starting with the SMS-style summary from the platform's original product
// brief. Deliberately independent of any specific delivery channel; see
// internal/vendor for that.
package format

import (
	"fmt"
	"strings"
)

// SalesResult mirrors shop_analytics's job.SalesResult wire shape (money
// fields as decimal strings, per shop_platform/money's JSON encoding).
// Duplicated rather than imported from shop_analytics — this package only
// needs the shape, not a build-time dependency on an unrelated service's
// internal packages.
type SalesResult struct {
	Total      string              `json:"total"`
	Shop       string              `json:"shop"`
	OrderCount int                 `json:"order_count"`
	AOV        string              `json:"aov"`
	Collective CollectiveBreakdown `json:"collective"`
}

type CollectiveBreakdown struct {
	Total    string             `json:"total"`
	Partners []PartnerBreakdown `json:"partners"`
}

type PartnerBreakdown struct {
	Name  string `json:"name"`
	Sales string `json:"sales"`
}

// AnalyticsResultPayload mirrors shop_analytics's job.Result wire shape.
type AnalyticsResultPayload struct {
	TenantID string `json:"tenant_id"`
	Job      string `json:"job"`
	Period   struct {
		Start string `json:"start"`
		End   string `json:"end"`
		Label string `json:"label"`
	} `json:"period"`
	Sales *SalesResult `json:"sales,omitempty"`
}

// SalesChannelBreakdown renders the example summary from the original
// product brief:
//
//	DevMoto Daily — Sep 11
//
//	Total Sales: $10,254
//
//	Shop:
//	$6,842
//	66.7%
//
//	Collective:
//	$3,412
//	33.3%
//
//	Collective Breakdown:
//	E-Moto Co        $1,282
//	...
//
//	Orders: 45
//	AOV: $227.87
//
// tenantDisplayName is passed in rather than looked up here — this package
// has no DB dependency of its own. Falls back to tenantID if no display
// name is available.
func SalesChannelBreakdown(tenantDisplayName string, payload AnalyticsResultPayload) (string, error) {
	if payload.Sales == nil {
		return "", fmt.Errorf("format: sales.channel.breakdown payload has no sales data")
	}
	s := payload.Sales

	total, err := parseCents(s.Total)
	if err != nil {
		return "", fmt.Errorf("format: parse total: %w", err)
	}
	shop, err := parseCents(s.Shop)
	if err != nil {
		return "", fmt.Errorf("format: parse shop: %w", err)
	}
	collective, err := parseCents(s.Collective.Total)
	if err != nil {
		return "", fmt.Errorf("format: parse collective total: %w", err)
	}

	var shopPct, collectivePct float64
	if total > 0 {
		shopPct = float64(shop) / float64(total) * 100
		collectivePct = float64(collective) / float64(total) * 100
	}

	var b strings.Builder
	fmt.Fprintf(&b, "%s — %s\n\n", tenantDisplayName, payload.Period.Label)
	fmt.Fprintf(&b, "Total Sales: %s\n\n", formatDollars(total))
	fmt.Fprintf(&b, "Shop:\n%s\n%.1f%%\n\n", formatDollars(shop), shopPct)
	fmt.Fprintf(&b, "Collective:\n%s\n%.1f%%\n\n", formatDollars(collective), collectivePct)

	if len(s.Collective.Partners) > 0 {
		b.WriteString("Collective Breakdown:\n")
		for _, p := range s.Collective.Partners {
			amt, err := parseCents(p.Sales)
			if err != nil {
				return "", fmt.Errorf("format: parse partner %q sales: %w", p.Name, err)
			}
			fmt.Fprintf(&b, "%s  %s\n", p.Name, formatDollars(amt))
		}
		b.WriteString("\n")
	}

	aov, err := parseCents(s.AOV)
	if err != nil {
		return "", fmt.Errorf("format: parse aov: %w", err)
	}
	fmt.Fprintf(&b, "Orders: %d\n", s.OrderCount)
	fmt.Fprintf(&b, "AOV: %s", formatDollars(aov))

	return b.String(), nil
}

// parseCents parses a "699.00"-style decimal string into integer cents,
// without depending on shop_platform/money (see the package doc comment).
func parseCents(s string) (int64, error) {
	if s == "" {
		return 0, nil
	}
	whole, frac, found := strings.Cut(s, ".")
	if !found {
		frac = "00"
	}
	if len(frac) < 2 {
		frac += strings.Repeat("0", 2-len(frac))
	}
	frac = frac[:2]

	var wholeVal, fracVal int64
	if _, err := fmt.Sscanf(whole, "%d", &wholeVal); err != nil {
		return 0, err
	}
	if _, err := fmt.Sscanf(frac, "%d", &fracVal); err != nil {
		return 0, err
	}
	if wholeVal < 0 {
		return wholeVal*100 - fracVal, nil
	}
	return wholeVal*100 + fracVal, nil
}

// formatDollars renders cents as "$10,254" when the value is a whole
// dollar amount, or "$227.87" when it carries cents — matching the
// original brief's example exactly (whole totals shown without ".00").
func formatDollars(cents int64) string {
	neg := cents < 0
	if neg {
		cents = -cents
	}
	dollars := cents / 100
	remainder := cents % 100

	whole := formatThousands(dollars)

	var s string
	if remainder == 0 {
		s = fmt.Sprintf("$%s", whole)
	} else {
		s = fmt.Sprintf("$%s.%02d", whole, remainder)
	}
	if neg {
		return "-" + s
	}
	return s
}

func formatThousands(n int64) string {
	s := fmt.Sprintf("%d", n)
	if len(s) <= 3 {
		return s
	}
	var parts []string
	for len(s) > 3 {
		parts = append([]string{s[len(s)-3:]}, parts...)
		s = s[:len(s)-3]
	}
	parts = append([]string{s}, parts...)
	return strings.Join(parts, ",")
}

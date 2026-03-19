package price

import (
	"fmt"
	"strings"
)

// ExportFormat specifies the target ledger dialect for price export.
type ExportFormat string

const (
	FormatLedger    ExportFormat = "ledger"
	FormatHLedger   ExportFormat = "hledger"
	FormatBeancount ExportFormat = "beancount"
)

// IsValidExportFormat reports whether f is a known export format.
func IsValidExportFormat(f ExportFormat) bool {
	switch f {
	case FormatLedger, FormatHLedger, FormatBeancount:
		return true
	default:
		return false
	}
}

// quoteName wraps a commodity name in double quotes when it contains whitespace.
// Both Ledger and hLedger accept quoted commodity names in that case.
// Beancount does not support quoted names, so callers must not apply this to
// beancount output.
func quoteName(name string) string {
	if strings.ContainsAny(name, " \t") {
		return `"` + name + `"`
	}
	return name
}

// FormatPrices renders a slice of Price values as plain text in the requested
// ledger dialect.  The input slice is assumed to already be in the desired sort
// order; FormatPrices does not reorder the entries.
//
// Supported formats and their line patterns:
//
//   - ledger:    P YYYY/MM/DD HH:MM:SS BASE VALUE QUOTE
//   - hledger:   P YYYY-MM-DD BASE VALUE QUOTE
//   - beancount: YYYY-MM-DD price BASE VALUE QUOTE
func FormatPrices(prices []Price, format ExportFormat) (string, error) {
	var sb strings.Builder
	for _, p := range prices {
		switch format {
		case FormatLedger:
			fmt.Fprintf(&sb, "P %s %s %s %s\n",
				p.Date.Format("2006/01/02 15:04:05"),
				quoteName(p.CommodityName),
				p.Value.String(),
				quoteName(p.QuoteCommodity),
			)
		case FormatHLedger:
			fmt.Fprintf(&sb, "P %s %s %s %s\n",
				p.Date.Format("2006-01-02"),
				quoteName(p.CommodityName),
				p.Value.String(),
				quoteName(p.QuoteCommodity),
			)
		case FormatBeancount:
			fmt.Fprintf(&sb, "%s price %s %s %s\n",
				p.Date.Format("2006-01-02"),
				p.CommodityName,
				p.Value.String(),
				p.QuoteCommodity,
			)
		default:
			return "", fmt.Errorf("unknown export format: %q", format)
		}
	}
	return sb.String(), nil
}

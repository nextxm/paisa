package cmd

import (
	"fmt"
	"time"

	"github.com/ananthakumaran/paisa/internal/model/migration"
	"github.com/ananthakumaran/paisa/internal/model/price"
	"github.com/ananthakumaran/paisa/internal/utils"
	log "github.com/sirupsen/logrus"
	"github.com/spf13/cobra"
)

var exportFormat string
var exportBase string
var exportQuote string
var exportFrom string
var exportTo string
var exportSource string

var exportCmd = &cobra.Command{
	Use:   "export",
	Short: "Export price history in ledger-family formats",
	Long: `Export the full price history from the database as plain-text price
directives in the selected dialect (ledger, hledger, or beancount).

The output is deterministically sorted by (date, commodity, quote, source) and
is suitable for appending directly to an existing journal file or redirecting
to a new file.

Examples:
  paisa export                            # full history, ledger format, stdout
  paisa export --format hledger           # hledger P directives
  paisa export --format beancount         # beancount price directives
  paisa export --base USD --quote INR     # only USD/INR prices
  paisa export --from 2024-01-01          # prices from 2024 onward
  paisa export --source journal           # only prices sourced from journal`,
	Run: func(cmd *cobra.Command, args []string) {
		format := price.ExportFormat(exportFormat)
		if !price.IsValidExportFormat(format) {
			log.Fatalf("invalid format %q: must be one of ledger, hledger, beancount", exportFormat)
		}

		filter := price.PriceFilter{
			Base:   exportBase,
			Quote:  exportQuote,
			Source: exportSource,
		}

		const dateLayout = "2006-01-02"
		if exportFrom != "" {
			t, err := time.Parse(dateLayout, exportFrom)
			if err != nil {
				log.Fatalf("invalid --from date %q: expected YYYY-MM-DD", exportFrom)
			}
			filter.From = t
		}
		if exportTo != "" {
			t, err := time.Parse(dateLayout, exportTo)
			if err != nil {
				log.Fatalf("invalid --to date %q: expected YYYY-MM-DD", exportTo)
			}
			filter.To = t
		}
		if !filter.From.IsZero() && !filter.To.IsZero() && filter.From.After(filter.To) {
			log.Fatalf("--from date must not be after --to date")
		}

		db, err := utils.OpenDB()
		if err != nil {
			log.Fatal(err)
		}

		if err := migration.RunMigrations(db); err != nil {
			log.Fatal(err)
		}

		prices, err := price.FindFiltered(db, filter)
		if err != nil {
			log.Fatalf("failed to query prices: %v", err)
		}

		text, err := price.FormatPrices(prices, format)
		if err != nil {
			log.Fatalf("failed to format prices: %v", err)
		}

		fmt.Print(text)
	},
}

func init() {
	rootCmd.AddCommand(exportCmd)
	exportCmd.Flags().StringVarP(&exportFormat, "format", "f", "ledger",
		"output format: ledger, hledger, or beancount")
	exportCmd.Flags().StringVar(&exportBase, "base", "",
		"filter by base commodity name (e.g. USD)")
	exportCmd.Flags().StringVar(&exportQuote, "quote", "",
		"filter by quote commodity (e.g. INR)")
	exportCmd.Flags().StringVar(&exportFrom, "from", "",
		"inclusive lower date bound (YYYY-MM-DD)")
	exportCmd.Flags().StringVar(&exportTo, "to", "",
		"inclusive upper date bound (YYYY-MM-DD)")
	exportCmd.Flags().StringVar(&exportSource, "source", "",
		"filter by price source (e.g. journal, com-yahoo)")
}

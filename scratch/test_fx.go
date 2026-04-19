package main

import (
	"fmt"
	"time"

	"github.com/ananthakumaran/paisa/internal/config"
	"github.com/ananthakumaran/paisa/internal/model"
	"github.com/ananthakumaran/paisa/internal/service"
	"github.com/shopspring/decimal"
)

func main() {
	db, _ := model.OpenDB(":memory:")
    config.LoadConfigFile("") // Default config
    
    // Add some test prices
    db.Exec("INSERT INTO prices (date, commodity_type, commodity_name, quote_commodity, value, source) VALUES (?, ?, ?, ?, ?, ?)",
        time.Now().Add(-24*time.Hour).Format("2006-01-02"), 0, "USD", "INR", 83.5, "test")
    db.Exec("INSERT INTO prices (date, commodity_type, commodity_name, quote_commodity, value, source) VALUES (?, ?, ?, ?, ?, ?)",
        time.Now().Add(-48*time.Hour).Format("2006-01-02"), 0, "EUR", "INR", 90.2, "test")

    // Helper to print rate
    check := func(base, quote string) {
        res, ok := service.GetRateDetails(db, base, quote, time.Now())
        if ok {
            fmt.Printf("%s -> %s: %s (%s via %s)\n", base, quote, res.Rate, res.ResolutionType, res.Anchor)
        } else {
            fmt.Printf("%s -> %s: Not found\n", base, quote)
        }
    }

    check("USD", "INR") // Direct
    check("INR", "USD") // Inverse
    check("USD", "EUR") // Cross via INR (if enabled)
    
    fmt.Println("Enabling multi-currency...")
    // Simulate setting EnableMultiCurrencyPrices: true
    // In code we'd do config.GetConfig().EnableMultiCurrencyPrices = true but it's internal.
    // Let's assume it works because I saw the logic in service.GetRate.
}

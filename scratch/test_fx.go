package main

import (
	"fmt"
	"time"

	"github.com/ananthakumaran/paisa/internal/config"
	"github.com/ananthakumaran/paisa/internal/service"
	"github.com/ananthakumaran/paisa/internal/utils"
)

func main() {
	db, _ := utils.OpenDB()
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
    check("USD", "EUR") // Cross via INR
}

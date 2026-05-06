package server

import (
"encoding/json"
"testing"

"github.com/ananthakumaran/paisa/internal/model/migration"
"github.com/ananthakumaran/paisa/internal/model/posting"
"github.com/glebarez/sqlite"
"github.com/shopspring/decimal"
"gorm.io/gorm"
"time"
)

func TestDBRoundtrip(t *testing.T) {
db, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{})
if err != nil {
t.Fatal(err)
}
migration.RunMigrations(db)

d, _ := time.Parse("2006-01-02", "2022-01-03")
db.Create(&posting.Posting{
TransactionID: "t1", Date: d, Payee: "Rent",
Account: "Expenses:Rent", Commodity: "INR",
Amount: decimal.NewFromFloat(20000), Quantity: decimal.NewFromFloat(20000),
})

var postings []posting.Posting
db.Find(&postings)

b, _ := json.Marshal(postings[0])
t.Logf("Posting from DB JSON: %s", string(b))

// Also check how amount marshals
t.Logf("Amount type: %T, value: %s", postings[0].Amount, postings[0].Amount.String())
ab, _ := json.Marshal(postings[0].Amount)
t.Logf("Amount JSON: %s", string(ab))
}

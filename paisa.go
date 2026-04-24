package main

import (
	"github.com/nextxm/paisa/cmd"
	"github.com/shopspring/decimal"
)

func main() {
	decimal.MarshalJSONWithoutQuotes = true
	cmd.Execute()
}

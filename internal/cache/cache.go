package cache

import (
	"github.com/nextxm/paisa/internal/accounting"
	"github.com/nextxm/paisa/internal/model/transaction"
	"github.com/nextxm/paisa/internal/prediction"
	"github.com/nextxm/paisa/internal/service"
)

func Clear() {
	service.ClearInterestCache()
	service.ClearPriceCache()
	service.ClearRateCache()
	accounting.ClearCache()
	prediction.ClearCache()
	transaction.ClearCache()
}

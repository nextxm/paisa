package stock

import (
	"encoding/json"
	"fmt"
	"io"
	"math/rand"
	"net/http"
	"sync"
	"time"

	"gorm.io/gorm"

	"github.com/shopspring/decimal"
	log "github.com/sirupsen/logrus"

	"github.com/ananthakumaran/paisa/internal/config"
	"github.com/ananthakumaran/paisa/internal/model/price"
	"github.com/ananthakumaran/paisa/internal/scraper/httpclient"
	"github.com/ananthakumaran/paisa/internal/utils"
)

// Compile-time check: YahooPriceProvider must satisfy price.PriceProvider.
var _ price.PriceProvider = (*YahooPriceProvider)(nil)

var UserAgents = []string{
	// Chrome
	"Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/133.0.0.0 Safari/537.36",
	"Mozilla/5.0 (Macintosh; Intel Mac OS X 10_15_7) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/133.0.0.0 Safari/537.36",
	"Mozilla/5.0 (X11; Linux x86_64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/133.0.0.0 Safari/537.36",

	// # Firefox
	"Mozilla/5.0 (Windows NT 10.0; Win64; x64; rv:135.0) Gecko/20100101 Firefox/135.0",
	"Mozilla/5.0 (Macintosh; Intel Mac OS X 14.7; rv:135.0) Gecko/20100101 Firefox/135.0",
	"Mozilla/5.0 (X11; Linux i686; rv:135.0) Gecko/20100101 Firefox/135.0",

	// # Safari
	"Mozilla/5.0 (Macintosh; Intel Mac OS X 14_7_4) AppleWebKit/605.1.15 (KHTML, like Gecko) Version/18.3 Safari/605.1.15",

	// # Edge
	"Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/133.0.0.0 Safari/537.36 Edg/131.0.2903.86",
}

type UserAgent struct {
	sync.Once
	name string
}

var agent UserAgent

func selectAgent() {
	agent.name = UserAgents[rand.Intn(len(UserAgents))]
}

type Quote struct {
	Close []*float64 `json:"close"`
}

type Indicators struct {
	Quote []Quote
}

type Meta struct {
	Currency string
}

type Result struct {
	Timestamp  []int64
	Indicators Indicators
	Meta       Meta
}

type Chart struct {
	Result []Result
}
type Response struct {
	Chart Chart
}

func GetHistory(ticker string, commodityName string) ([]*price.Price, error) {
	log.Info("Fetching stock price history from Yahoo")
	response, err := getTicker(ticker)
	if err != nil {
		return nil, err
	}

	var prices []*price.Price
	if len(response.Chart.Result) == 0 {
		return nil, fmt.Errorf("Failed to fetch data for %s, is the ticker valid?", ticker)
	}
	result := response.Chart.Result[0]
	nativeCurrency := result.Meta.Currency
	defaultCurrency := config.DefaultCurrency()
	needExchangePrice := !utils.IsCurrency(nativeCurrency)

	// Store stock prices in their native currency.
	prices = appendYahooPrices(prices, result, func(date time.Time, value float64) price.Price {
		return price.Price{
			Date:           date,
			CommodityType:  config.Stock,
			CommodityID:    ticker,
			CommodityName:  commodityName,
			Value:          decimal.NewFromFloat(value),
			QuoteCommodity: nativeCurrency,
		}
	})

	// When the native currency differs from the default currency, fetch and
	// store the exchange rate as a separate set of price entries so that the
	// market-price service can convert native prices to the default currency.
	if needExchangePrice {
		exchangeTicker := fmt.Sprintf("%s%s=X", nativeCurrency, defaultCurrency)
		exchangeResponse, err := getTicker(exchangeTicker)
		if err != nil {
			return nil, err
		}
		if len(exchangeResponse.Chart.Result) == 0 {
			return nil, fmt.Errorf("Failed to fetch exchange data for %s", exchangeTicker)
		}
		exchangeResult := exchangeResponse.Chart.Result[0]
		prices = appendYahooPrices(prices, exchangeResult, func(date time.Time, value float64) price.Price {
			return price.Price{
				Date:           date,
				CommodityType:  config.Stock,
				CommodityID:    exchangeTicker,
				CommodityName:  nativeCurrency,
				Value:          decimal.NewFromFloat(value),
				QuoteCommodity: defaultCurrency,
				Source:         "com-yahoo",
			}
		})
	}

	return prices, nil
}

func appendYahooPrices(prices []*price.Price, result Result, build func(time.Time, float64) price.Price) []*price.Price {
	if len(result.Indicators.Quote) == 0 {
		return prices
	}

	closes := result.Indicators.Quote[0].Close
	for i, timestamp := range result.Timestamp {
		if i >= len(closes) || closes[i] == nil {
			continue
		}

		p := build(time.Unix(timestamp, 0), *closes[i])
		prices = append(prices, &p)
	}

	return prices
}

func getTicker(ticker string) (*Response, error) {
	url := fmt.Sprintf("https://query2.finance.yahoo.com/v8/finance/chart/%s?interval=1d&range=50y", ticker)
	req, err := http.NewRequest("GET", url, nil)
	if err != nil {
		return nil, err
	}

	agent.Do(func() { selectAgent() })
	req.Header.Add("User-Agent", agent.name)

	resp, err := httpclient.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	respBytes, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}

	var response Response
	err = json.Unmarshal(respBytes, &response)
	if err != nil {
		return nil, err
	}

	return &response, nil
}

type YahooPriceProvider struct {
}

func (p *YahooPriceProvider) Code() string {
	return "com-yahoo"
}

func (p *YahooPriceProvider) Label() string {
	return "Yahoo Finance"
}

func (p *YahooPriceProvider) Description() string {
	return "Supports a large set of stocks, ETFs, mutual funds, currencies, bonds, commodities, and cryptocurrencies. Prices are stored in their native currency; exchange rates are saved separately and used for automatic conversion to the default currency."
}

func (p *YahooPriceProvider) RateLimit() price.ProviderRateLimit {
	return price.ProviderRateLimit{MaxConcurrentRequests: 2}
}

func (p *YahooPriceProvider) AutoCompleteFields() []price.AutoCompleteField {
	return []price.AutoCompleteField{
		{Label: "Ticker", ID: "ticker", Help: "Stock ticker symbol, can be located on Yahoo's website. For example, AAPL is the ticker symbol for Apple Inc. (AAPL)", InputType: "text"},
	}
}

func (p *YahooPriceProvider) AutoComplete(db *gorm.DB, field string, filter map[string]string) []price.AutoCompleteItem {
	return []price.AutoCompleteItem{}
}

func (p *YahooPriceProvider) ClearCache(db *gorm.DB) {
}

func (p *YahooPriceProvider) GetPrices(code string, commodityName string, since time.Time) ([]*price.Price, error) {
	prices, err := GetHistory(code, commodityName)
	if err != nil {
		return nil, err
	}
	return price.FilterSince(prices, since), nil
}

func (p *YahooPriceProvider) GetPricesBatch(codes []string, commodityNames []string) (map[string][]*price.Price, error) {
	return price.GetPricesBatchSequentially(p, codes, commodityNames)
}

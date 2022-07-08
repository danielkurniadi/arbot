package exchange

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"regexp"
	"time"

	go1inch "github.com/jon4hz/go-1inch"
	"github.com/shopspring/decimal"
)

// Seems like from reverse engineer, 1inch uses coin gecko
// as oracle to provide swap price rate.
//
// See: https://www.coingecko.com/en/api/documentation
var tokenToCoinGeckoId = map[string]string{
	"BTC":  "bitcoin",
	"ETH":  "ethereum",
	"USDC": "usd",
	"USDT": "usd",
	"DAI":  "usd",
}

func convertToCoinGeckoTokenId(token string) string {
	if tokenId, has := tokenToCoinGeckoId[token]; has {
		return tokenId
	}
	return token
}

// Docs: https://docs.1inch.io/
type oneInchExchange struct {
	Config
	client go1inch.Client
}

func New1InchExchange(config Config) *oneInchExchange {
	return &oneInchExchange{
		Config: config,
		client: *go1inch.NewClient(),
	}
}

func (exc *oneInchExchange) Name() string { return "1inch" }

func (exc *oneInchExchange) Ping(ctx context.Context) error {
	_, _, err := exc.client.Healthcheck(ctx, "eth")
	return err
}

func (exc *oneInchExchange) Fees() decimal.Decimal {
	return decimal.NewFromFloat(exc.Config.Fees)
}

func (exc *oneInchExchange) PriceStream(symbol string, interval time.Duration) (<-chan Price, error) {
	priceChan := make(chan Price, maxDefautPriceBuffer)
	timer := time.NewTicker(interval)

	go func(timer *time.Ticker, priceChan chan<- Price) {
		for range timer.C { // wake up every interval
			price, err := exc.fetchPrice(symbol)

			if err != nil {
				log.Println("debug: fetch price 1inch failed", err)
				continue
			}

			// Push price after polling
			priceChan <- price
		}
	}(timer, priceChan)

	return priceChan, nil
}

const coinGeckoPriceURL = "https://api.coingecko.com/api/v3/simple/price"

func (exc *oneInchExchange) fetchPrice(symbol string) (price Price, err error) {
	base, quote := exc.parseBaseAndQuote(symbol)

	var payload = make(map[string]map[string]float64)
	request, _ := http.NewRequest(http.MethodGet, coinGeckoPriceURL, nil)

	queryParams := request.URL.Query()
	queryParams.Add("ids", base)
	queryParams.Add("vs_currencies", quote)

	// build query params: e.g. "?ids=ethereum&vs_currency=usd"
	request.URL.RawQuery = queryParams.Encode()

	response, err := http.DefaultClient.Do(request)
	if err != nil {
		return Price{}, err
	}

	if err := json.NewDecoder(response.Body).Decode(&payload); err != nil {
		return Price{}, fmt.Errorf("%w: %v", ErrPriceInvalidData, err)
	}

	value := decimal.NewFromFloat(payload[base][quote])

	return Price{
		Ask: value,
		Bid: value,
	}, nil
}

func (exc *oneInchExchange) parseBaseAndQuote(symbol string) (base, quote string) {
	pattern := regexp.MustCompile(regexpTicker)

	result := make(map[string]string)
	match := pattern.FindStringSubmatch(symbol)

	for i, name := range pattern.SubexpNames() {
		if i != 0 && name != "" {
			result[name] = match[i]
		}
	}

	base = convertToCoinGeckoTokenId(result["base"])
	quote = convertToCoinGeckoTokenId(result["quote"])

	return base, quote
}

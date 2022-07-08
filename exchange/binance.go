package exchange

import (
	"context"
	"regexp"
	"time"

	"github.com/adshao/go-binance/v2"
	"github.com/shopspring/decimal"
)

const (
	binanceTickerFmt      = "$base$quote" // symbol without delimiter
	binanceStatusTradable = "TRADING"     // binance api: status for symbol
	maxDefautPriceBuffer  = 1000
)

// Docs: https://binance-docs.github.io/
type binanceExchange struct {
	Config
	client *binance.Client
}

func NewBinanceExchange(config Config) Exchange {
	binance.UseTestnet = config.Testing

	client := binance.NewClient(
		config.APIAccessKey,
		config.APISecretKey,
	)

	return &binanceExchange{
		Config: config,
		client: client,
	}
}

func (exc *binanceExchange) Name() string { return "binance" }

func (exc *binanceExchange) Ping(ctx context.Context) error {
	err := exc.client.NewPingService().Do(ctx)
	return err
}

type binancePriceStream struct {
	priceChan chan Price      // send price to subscriber
	stopChan  chan<- struct{} // send stop signal
}

func (m *binancePriceStream) Close() error {
	// we already stopped the stream
	_, active := <-m.priceChan
	if !active {
		return nil
	}

	// we stop the stream
	m.stopChan <- struct{}{}
	close(m.priceChan)

	return nil
}

func (exc *binanceExchange) PriceStream(symbol string, _ time.Duration) (<-chan Price, error) {
	symbol, err := exc.formatSymbol(symbol)
	if err != nil {
		// invalid ticker format
		return nil, err
	}

	stream, err := exc.webSocketPriceStream(symbol)
	if err != nil {
		return nil, err
	}

	return stream.priceChan, nil
}

func (exc *binanceExchange) Fees() decimal.Decimal {
	return decimal.NewFromFloat(exc.Config.Fees)
}

func (exc *binanceExchange) formatSymbol(symbol string) (string, error) {
	pattern := regexp.MustCompile(regexpTicker)
	result := make([]byte, 0, len(symbol))

	// use regexp groups to capture {base} and {quote} labels
	submatches := pattern.FindAllSubmatchIndex([]byte(symbol), -1)

	// contruct symbol string with format: "{base}{quote}"
	for _, submatch := range submatches {
		result = pattern.ExpandString(
			result, binanceTickerFmt,
			symbol, submatch,
		)
	}

	return string(result), nil
}

func (exc *binanceExchange) webSocketPriceStream(symbol string) (*binancePriceStream, error) {
	priceChan := make(chan Price, maxDefautPriceBuffer)

	// Initiate new websocket connection.
	_, stopChan, err := binance.WsBookTickerServe(symbol,
		// On handle websocket event.
		func(event *binance.WsBookTickerEvent) {
			ask, err := decimal.NewFromString(event.BestAskPrice)
			if err != nil { // bad data
				return
			}

			bid, err := decimal.NewFromString(event.BestBidPrice)
			if err != nil { // bad data
				return
			}

			price := Price{
				Ask: ask,
				Bid: bid,
			}

			select {
			case priceChan <- price:

			default:

			}
		},
		func(err error) {
			// TODO: do something
		})

	if err != nil {
		return nil, err
	}

	stream := &binancePriceStream{
		priceChan: priceChan,
		stopChan:  stopChan,
	}

	return stream, nil
}

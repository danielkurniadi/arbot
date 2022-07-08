package exchange

import (
	"context"
	"errors"
	"time"

	"github.com/shopspring/decimal"
)

const (
	// e.g. {base}:{quote}, {base}-{quote}. {base}_{quote}
	regexpTicker = `^(?P<base>[A-Za-z]+)[:\-_]{0,1}(?P<quote>[A-Za-z]+)$`
)

var (
	ErrInvalidSymbol    = errors.New("arbot: invalid ticker symbol format")
	ErrPriceInvalidData = errors.New("arbot: price receive bad price data")
)

type Price struct {
	// spot orderbook price
	Ask decimal.Decimal
	Bid decimal.Decimal
}

type Config struct {
	Name         string  `yaml:"name"`
	Testing      bool    `yaml:"testing"`
	Fees         float64 `yaml:"fees"` // per trx
	APIAccessKey string  `yaml:"apiAccessKey"`
	APISecretKey string  `yaml:"apiSecretKey"`

	// support for other auths
	// ...
}

type Exchange interface {
	Name() string
	Ping(ctx context.Context) error
	Fees() decimal.Decimal
	PriceStream(symbol string, interval time.Duration) (<-chan Price, error)
}

func NewExchange(config Config) Exchange {
	switch config.Name {
	case "binance":
		return NewBinanceExchange(config)
	case "1inch":
		return New1InchExchange(config)
	default:
		panic("arbot: unknown exchange: " + config.Name)
	}
}

package strategy

import (
	"context"
	"errors"
	"fmt"
	"log"
	"strings"
	"sync"
	"time"

	"github.com/iqDF/arbot/exchange"
	"github.com/shopspring/decimal"
)

var (
	errInitPriceStream = errors.New("arbot: init price stream err")
)

type Config struct {
	Name        string              `yaml:"name"`
	Debug       bool                `yaml:"debug"`
	Interval    time.Duration       `yaml:"interval"`
	Exchanges   []exchange.Exchange `yaml:"-"`
	TradingPair string              `yaml:"tradingPair"` // symbol ticker
	Slipage     float64             `yaml:"slipage"`
}

type Strategy struct {
	Config
	// Cross-exchanges arbitrage
	exchangeA exchange.Exchange
	exchangeB exchange.Exchange
}

func NewStrategy(config Config) *Strategy {
	if len(config.Exchanges) < 2 {
		panic("arbot: cross-exchange single asset requires 2 exchanges!")
	}

	return &Strategy{
		Config:    config,
		exchangeA: config.Exchanges[0],
		exchangeB: config.Exchanges[1],
	}
}

// materialized view of the price should
// provide the updated price from feed.
type priceView struct {
	// mutex to protect price
	exchange string
	mu       sync.RWMutex
	// price suppose to be most recently
	// updated.
	price       exchange.Price
	initialized bool
	signalReady chan struct{}
}

func newPriceView(exchange string) *priceView {
	return &priceView{
		exchange:    exchange,
		initialized: false,
		signalReady: make(chan struct{}, 1),
	}
}

func (mv *priceView) Price() exchange.Price {
	mv.mu.RLock()
	defer mv.mu.RUnlock()

	priceCopy := mv.price
	return priceCopy
}

func (mv *priceView) Update(price exchange.Price) {
	mv.mu.Lock()
	defer mv.mu.Unlock()

	if !mv.initialized {
		mv.signalReady <- struct{}{}
		mv.initialized = true
	}

	mv.price = price
}

func (strat *Strategy) Run(ctx context.Context) error {
	viewA, viewB, err := strat.initDualPriceViews(strat.TradingPair)
	if err != nil {
		return err
	}

	var ticker = time.NewTicker(strat.Interval)
	for { // should be CPU safe.
		select {
		case <-ctx.Done():
			return ctx.Err()

		case <-ticker.C:
			priceA := viewA.Price()
			priceB := viewB.Price()

			// arbitrage buy from exchange A and sell to exchange B
			profitAB := strat.calcAdjustedProfit(priceB.Ask, priceA.Bid)
			planAB := ArbitragePlan{
				tradingPair:  strat.TradingPair,
				timestamp:    time.Now(),
				buyExchange:  strat.exchangeA.Name(),
				buyPrice:     priceA.Bid,
				sellExchange: strat.exchangeB.Name(),
				sellPrice:    priceB.Ask,
				profit:       profitAB,
				slipage:      strat.Slipage,
			}

			if profitAB.GreaterThan(decimal.Zero) {
				log.Println(planAB.Format(), "profit: yes |")
			} else {
				log.Println(planAB.Format(), "profit: no  |")
			}

			// arbitrage buy from exchange B and sell to exchange A
			profitBA := strat.calcAdjustedProfit(priceA.Ask, priceB.Bid)
			planBA := ArbitragePlan{
				tradingPair:  strat.TradingPair,
				timestamp:    time.Now(),
				buyExchange:  strat.exchangeB.Name(),
				buyPrice:     priceB.Bid,
				sellExchange: strat.exchangeA.Name(),
				sellPrice:    priceA.Ask,
				profit:       profitBA,
				slipage:      strat.Slipage,
			}

			if profitBA.GreaterThan(decimal.Zero) {
				log.Println(planBA.Format(), "profit: yes |")
			} else {
				log.Println(planBA.Format(), "profit: no  |")
			}
		}
	}
}

func (strat *Strategy) calcAdjustedProfit(sell, buy decimal.Decimal) decimal.Decimal {
	sellSlip := decimal.NewFromFloat(1.0 - strat.Slipage)
	sellFees := sell.Mul(strat.exchangeA.Fees())

	buySlip := decimal.NewFromFloat(1.0 + strat.Slipage)
	buyFees := buy.Mul(strat.exchangeB.Fees())

	sellAdj := sell.Mul(sellSlip) // sell x (1 - slip)
	buyAdj := buy.Mul(buySlip)    // buy x (1 + slip)

	// profit: sellAdj - buyAdj - sellFee - buyFee
	return (sellAdj.Sub(buyAdj)).Sub(sellFees.Add(buyFees))
}

func (strat *Strategy) initDualPriceViews(symbol string) (*priceView, *priceView, error) {
	interval := strat.Interval

	// initialize price stream and materialized view
	viewA := newPriceView(strat.exchangeA.Name())
	streamA, err := strat.exchangeA.PriceStream(symbol, interval)
	if err != nil {
		return nil, nil, fmt.Errorf("%w %v", errInitPriceStream, err)
	}

	// initialize price stream and materialized view
	viewB := newPriceView(strat.exchangeB.Name())
	streamB, err := strat.exchangeB.PriceStream(symbol, interval)
	if err != nil {
		return nil, nil, fmt.Errorf("%w %v", errInitPriceStream, err)
	}

	// run a new go routine to consume price feeds
	go strat.consumePriceStream(streamA, viewA)
	go strat.consumePriceStream(streamB, viewB)

	// wait until both are populated / ready
	<-viewA.signalReady
	<-viewB.signalReady

	return viewA, viewB, nil
}

func (strat *Strategy) consumePriceStream(stream <-chan exchange.Price, priceMV *priceView) {
	// keep looping forever or until
	// the price stream is closed by sender.
	for price := range stream {
		priceMV.Update(price)
	}
}

// arbitrage plan helps formatting the plan on how
// to do arbitrage into tabulated printable string.
type ArbitragePlan struct {
	tradingPair  string
	timestamp    time.Time
	buyExchange  string
	sellExchange string
	buyPrice     decimal.Decimal
	sellPrice    decimal.Decimal
	profit       decimal.Decimal
	slipage      float64
	totalFee     float64
}

func (p ArbitragePlan) Header() string {
	// 2022-07-08T21:44:16+08:00
	return fmt.Sprintf(`%-20s | %-15s | %-15s | %-15s | %-15s | %-15s | %-15s | %-10s | %-10s | Notify?     |`,
		"Timestamp",
		"Trading Pair",
		"Buy Exchange",
		"Buy Price",
		"Sell Exchange",
		"Sell Price",
		"Profit",
		"Slipage",
		"Total fee",
	)
}

func (p ArbitragePlan) Divider() string {
	return strings.Repeat("-", 170)
}

func (p ArbitragePlan) Format() string {
	return fmt.Sprintf(` | %-15s | %-15s | %-15.4f | %-15s | %-15.4f | %-15.4f | %10.4f | %10.4f |`,
		p.tradingPair,
		p.buyExchange,
		p.buyPrice.InexactFloat64(),
		p.sellExchange,
		p.sellPrice.InexactFloat64(),
		p.profit.InexactFloat64(),
		p.slipage,
		p.totalFee,
	)
}

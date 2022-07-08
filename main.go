package main

import (
	"context"
	"flag"
	"fmt"
	"io/ioutil"
	"log"
	"os"
	"os/signal"
	"syscall"

	"github.com/iqDF/arbot/exchange"
	"github.com/iqDF/arbot/strategy"
	"gopkg.in/yaml.v2"
)

type AppConfig struct {
	Exchanges  []exchange.Config `yaml:"exchanges"`
	Strategies []strategy.Config `yaml:"strategies"`
}

var (
	configPath string = "config.yml"
)

func init() {
	flag.StringVar(&configPath, "config", "config.yml", "Path to arbot yaml config")
	flag.Parse()
}

func main() {
	// Read application config
	//
	var appConfig AppConfig

	yamlFile, err := ioutil.ReadFile(configPath)
	if err != nil {
		log.Println("arbot: cannot read yaml config file", err, configPath)
		return
	}

	if err = yaml.Unmarshal(yamlFile, &appConfig); err != nil {
		log.Fatalln("arbot: bad yaml config:", err)
		return
	}

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	// Init exchange connectors
	//
	exchanges := make([]exchange.Exchange, 0)

	log.Println("arbot: setup and init exchange connectors ...")

	for _, excConf := range appConfig.Exchanges {
		exchange := exchange.NewExchange(excConf)
		exchanges = append(exchanges, exchange)
	}

	// DEBUG pretty logging
	log.Printf("arbot: setup and init strategy runners ...\n\n")

	fmt.Println(strategy.ArbitragePlan{}.Header())
	fmt.Println(strategy.ArbitragePlan{}.Divider())

	// Init strategy runners
	//
	for _, stratConf := range appConfig.Strategies {
		stratConf.Exchanges = exchanges
		strat := strategy.NewStrategy(stratConf)

		go strat.Run(ctx)
	}

	sigchan := make(chan os.Signal, 1)
	signal.Notify(sigchan, syscall.SIGINT, syscall.SIGTERM)

	// Waiting for CTRL-C or CTRL-Z
	<-sigchan
}

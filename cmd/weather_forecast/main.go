package main

import (
	"context"
	"fmt"

	"github.com/angas/solarplant-go/config"
	"github.com/angas/solarplant-go/tibber"
)

func main() {
	config := config.Load()
	tibber := tibber.New(config.Tibber.ApiToken, config.Tibber.HomeId)
	res, err := tibber.GetEnergyPrices(context.Background())
	if err != nil {
		panic(err)
	}

	for _, p := range res {
		fmt.Printf("Date: %s, Hour: %d, Price: %f, Tax: %f\n",
			p.StartsAt.Format("2006-01-02"), p.StartsAt.Hour(), p.Energy, p.Tax)
	}
}

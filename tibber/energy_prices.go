package tibber

import (
	"context"
	"time"
)

type priceInfo struct {
	StartsAt string  `json:"startsAt"`
	Energy   float64 `json:"energy"`
	Tax      float64 `json:"tax"`
}

type priceInfoResponse struct {
	CurrentSubscription struct {
		PriceInfo struct {
			Today    []priceInfo `json:"today"`
			Tomorrow []priceInfo `json:"tomorrow"`
		} `json:"priceInfo"`
	} `json:"currentSubscription"`
}

type tibberEnergyPrice struct {
	StartsAt time.Time
	Energy   float64
	Tax      float64
}

func (t *Tibber) GetEnergyPrices(ctx context.Context) ([]tibberEnergyPrice, error) {
	query := `
		currentSubscription {
			priceInfo {
				today { startsAt energy tax } 
				tomorrow { startsAt energy tax }
			}
		}`

	body, err := doQuery[priceInfoResponse](ctx, t.ApiToken, t.HomeId, query)
	if err != nil {
		return nil, err
	}

	todayAndTomorrow := append(
		body.Data.Viewer.Home.CurrentSubscription.PriceInfo.Today,
		body.Data.Viewer.Home.CurrentSubscription.PriceInfo.Tomorrow...)

	prices := make([]tibberEnergyPrice, 0, len(todayAndTomorrow))

	for _, price := range todayAndTomorrow {
		startsAt, err := time.Parse(time.RFC3339, price.StartsAt)
		if err != nil {
			return nil, err
		}
		prices = append(prices, tibberEnergyPrice{StartsAt: startsAt.UTC(), Energy: price.Energy, Tax: price.Tax})
	}

	return prices, nil
}

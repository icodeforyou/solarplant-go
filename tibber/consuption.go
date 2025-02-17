package tibber

import (
	"fmt"
	"time"

	"golang.org/x/net/context"
)

type consumption struct {
	Consumption struct {
		Nodes []struct {
			From            string  `json:"from"`
			To              string  `json:"to"`
			Consumption     float64 `json:"consumption"`
			ConsumptionUnit string  `json:"consumptionUnit"`
			Cost            float64 `json:"cost"`
			UnitPrice       float64 `json:"unitPrice"`
			UnitPriceVAT    float64 `json:"unitPriceVAT"`
		} `json:"nodes"`
	} `json:"consumption"`
}

type tibberConsumption struct {
	StartsAt        time.Time
	Consumption     float64
	ConsumptionUnit string
	Cost            float64
	UnitPrice       float64
	UnitPriceVAT    float64
}

func (t *Tibber) GetConsumption(ctx context.Context, hours int) ([]tibberConsumption, error) {
	query := fmt.Sprintf(`
		consumption(resolution: HOURLY, last: %d) {
			nodes {
				from
				to
				consumption
				consumptionUnit
				cost
				unitPrice
				unitPriceVAT  
			}
		}`, hours)

	body, err := doQuery[consumption](ctx, t.ApiToken, t.HomeId, query)
	if err != nil {
		return nil, err
	}

	consumption := make([]tibberConsumption, 0, len(body.Data.Viewer.Home.Consumption.Nodes))
	for _, p := range body.Data.Viewer.Home.Consumption.Nodes {
		from, err := time.Parse(time.RFC3339, p.From)
		if err != nil {
			return nil, err
		}

		consumption = append(consumption, tibberConsumption{
			StartsAt:        from.UTC(),
			Consumption:     p.Consumption,
			ConsumptionUnit: p.ConsumptionUnit,
			Cost:            p.Cost,
			UnitPrice:       p.UnitPrice,
			UnitPriceVAT:    p.UnitPriceVAT,
		})
	}

	return consumption, nil
}

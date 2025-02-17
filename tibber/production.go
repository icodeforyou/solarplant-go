package tibber

import (
	"context"
	"fmt"
	"time"
)

type productionResponse struct {
	Production struct {
		Nodes []struct {
			From           string  `json:"from"`
			To             string  `json:"to"`
			Production     float64 `json:"production"`
			ProductionUnit string  `json:"productionUnit"`
			Profit         float64 `json:"profit"`
			UnitPrice      float64 `json:"unitPrice"`
			UnitPriceVAT   float64 `json:"unitPriceVAT"`
		} `json:"nodes"`
	} `json:"production"`
}

type tibberProduction struct {
	StartsAt       time.Time
	Production     float64
	ProductionUnit string
	Profit         float64
	UnitPrice      float64
	UnitPriceVAT   float64
}

func (t *Tibber) GetProduction(ctx context.Context, hours int) ([]tibberProduction, error) {
	query := fmt.Sprintf(`query {
		viewer {
			home(id:"%s") {
				production(resolution: HOURLY, last: %d) {
					nodes {
						from
          	to
          	production
          	productionUnit
          	profit
          	unitPrice
          	unitPriceVAT
					}
				}
			}
		}
	}`, t.HomeId, hours)

	body, err := doQuery[productionResponse](ctx, t.ApiToken, t.HomeId, query)
	if err != nil {
		return nil, err
	}

	production := make([]tibberProduction, 0, len(body.Data.Viewer.Home.Production.Nodes))
	for _, p := range body.Data.Viewer.Home.Production.Nodes {
		from, err := time.Parse(time.RFC3339, p.From)
		if err != nil {
			return nil, err
		}

		production = append(production, tibberProduction{
			StartsAt:       from.UTC(),
			Production:     p.Production,
			ProductionUnit: p.ProductionUnit,
			Profit:         p.Profit,
			UnitPrice:      p.UnitPrice,
			UnitPriceVAT:   p.UnitPriceVAT,
		})
	}

	return production, nil
}

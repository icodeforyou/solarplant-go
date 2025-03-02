package nordpool

import (
	"context"
	"encoding/json"
	"fmt"
	"math"
	"net/http"
	"slices"
	"time"

	"github.com/angas/solarplant-go/hours"
	"github.com/angas/solarplant-go/types"
)

type nordpoolData struct {
	Version          int `json:"version"`
	MultiAreaEntries []struct {
		DeliveryStart time.Time          `json:"deliveryStart"`
		DeliveryEnd   time.Time          `json:"deliveryEnd"`
		EntryPerArea  map[string]float64 `json:"entryPerArea"`
	} `json:"multiAreaEntries"`
}

type Nordpool struct {
	area string
}

func New(area string) Nordpool {
	return Nordpool{area: area}
}

func (n Nordpool) GetEnergyPrices(ctx context.Context) ([]types.EnergyPrice, error) {

	t := time.Now()
	today, err := n.getEnergyPrices(ctx, t)
	if err != nil {
		return nil, fmt.Errorf("failed to fetch prices from nordpool for today: %w", err)
	}

	if t.Before(time.Date(t.Year(), t.Month(), t.Day(), 14, 15, 0, 0, time.Local)) {
		return today, nil
	}

	tomorrow, err := n.getEnergyPrices(ctx, t.AddDate(0, 0, 1))
	if err != nil {
		return nil, fmt.Errorf("failed to fetch prices from nordpool for tomorrow: %w", err)
	}

	return append(today, tomorrow...), nil
}

func (n Nordpool) getEnergyPrices(ctx context.Context, date time.Time) ([]types.EnergyPrice, error) {
	url := fmt.Sprintf("%s/api/DayAheadPrices?date=%s&market=DayAhead&deliveryArea=%s&currency=SEK",
		"https://dataportal-api.nordpoolgroup.com",
		date.Format("2006-01-02"),
		n.area)

	req, err := http.NewRequestWithContext(ctx, "GET", url, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to create nordpool request: %w", err)
	}
	client := &http.Client{}
	resp, err := client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("failed to fetch nordpool prices: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode == http.StatusNotFound {
		return []types.EnergyPrice{}, nil
	}

	if resp.StatusCode == http.StatusNoContent {
		return []types.EnergyPrice{}, nil
	}

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("unexpected status code from nordpool: %d", resp.StatusCode)
	}

	var data nordpoolData
	if err := json.NewDecoder(resp.Body).Decode(&data); err != nil {
		return nil, fmt.Errorf("failed to decode nordpool response: %w", err)
	}

	prices := make([]types.EnergyPrice, 0)
	for _, entry := range data.MultiAreaEntries {
		hour := hours.FromTime(entry.DeliveryStart)
		if slices.ContainsFunc(prices, func(p types.EnergyPrice) bool { return p.Hour == hour }) {
			continue
		}
		price, ok := entry.EntryPerArea[n.area]
		if ok {
			prices = append(prices, types.EnergyPrice{
				Hour:  hour,
				Price: normalizePrice(price),
			})
		}
	}

	return prices, nil
}

func normalizePrice(price float64) float64 {
	precision := math.Pow(10, float64(4))
	return math.Round(price*precision/1e3) / precision
}

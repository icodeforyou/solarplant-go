package elprisetjustnu

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"time"

	"github.com/angas/solarplant-go/hours"
	"github.com/angas/solarplant-go/types"
)

type rawPrice struct {
	SEKPerKWh float64   `json:"SEK_per_kWh"`
	EURPerKWh float64   `json:"EUR_per_kWh"`
	EXR       float64   `json:"EXR"`
	TimeStart time.Time `json:"time_start"`
	TimeEnd   time.Time `json:"time_end"`
}

type ElPrisetJustNu struct {
	area string
}

func New(area string) ElPrisetJustNu {
	return ElPrisetJustNu{area: area}
}

func (e ElPrisetJustNu) GetEnergyPrices(ctx context.Context) ([]types.EnergyPrice, error) {
	now := time.Now()
	url := fmt.Sprintf("https://www.elprisetjustnu.se/api/v1/prices/%d/%02d-%02d_%s.json",
		now.Year(), now.Month(), now.Day(), e.area)

	req, err := http.NewRequestWithContext(ctx, "GET", url, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}
	client := &http.Client{}
	resp, err := client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("failed to fetch prices: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("unexpected status code: %d", resp.StatusCode)
	}

	var rawPrices []rawPrice
	if err := json.NewDecoder(resp.Body).Decode(&rawPrices); err != nil {
		return nil, fmt.Errorf("failed to decode response: %w", err)
	}

	prices := make([]types.EnergyPrice, 0, len(rawPrices))
	for _, raw := range rawPrices {
		prices = append(prices, types.EnergyPrice{
			Hour:  hours.FromTime(raw.TimeStart),
			Price: raw.SEKPerKWh,
		})
	}

	return prices, nil
}

package nordpool

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"log/slog"
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/angas/solarplant-go/hours"
	"github.com/angas/solarplant-go/types"
)

type Nordpool struct {
	area string
	page int16
}

func New(area string) Nordpool {
	return Nordpool{area: area}
}

func (n Nordpool) GetEnergyPrices(ctx context.Context) ([]types.EnergyPrice, error) {
	url := fmt.Sprintf("%s/marketdata/page/%d?currency=SEK", API_URL, n.page)

	slog.Default().Info("Fetching energy prices from Nordpool...", "url", url)

	req, _ := http.NewRequestWithContext(ctx, "GET", url, nil)
	client := http.Client{Timeout: 10 * time.Second}
	res, err := client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("error getting NordPool prices: %v", err)
	}
	defer res.Body.Close()

	body, err := io.ReadAll(res.Body)
	if err != nil {
		return nil, fmt.Errorf("error reading NordPool response body: %v", err)
	}

	var nordpool nordpool
	if err := json.Unmarshal(body, &nordpool); err != nil {
		return nil, fmt.Errorf("error unmarshaling NordPool json: %v", err)
	}

	result := make([]types.EnergyPrice, 0)
	for _, row := range nordpool.Data.Rows {
		for _, column := range row.Columns {
			if column.Name == n.area {
				t, err := time.Parse("2006-01-02T15:04:05", row.StartTime)
				if err != nil {
					return nil, fmt.Errorf("error parsing time: %v", err)
				}
				result = append(result, types.EnergyPrice{
					Hour:  hours.FromTime(t),
					Price: priceToFloat(column.Value),
				})
			}
		}
	}

	return result, nil
}

func priceToFloat(str string) float64 {
	noSpace := strings.ReplaceAll(str, " ", "")
	noComma := strings.ReplaceAll(noSpace, ",", ".")
	price, err := strconv.ParseFloat(noComma, 64)
	if err != nil {
		return 0
	}
	return price / 1e3
}

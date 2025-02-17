package tibber

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"
)

type queryRequest struct {
	Query string `json:"query"`
}

type queryResponse[T any] struct {
	Data struct {
		Viewer struct {
			Home T `json:"home"`
		} `json:"viewer"`
	} `json:"data"`
	Errors []struct {
		Message string   `json:"message"`
		Path    []string `json:"path"`
	} `json:"errors,omitempty"`
}

type Tibber struct {
	ApiToken string
	HomeId   string
}

func New(apiToken string, homeId string) *Tibber {
	return &Tibber{ApiToken: apiToken, HomeId: homeId}
}

func doQuery[T any](ctx context.Context, apiToken string, homeId string, innerQuery string) (*queryResponse[T], error) {
	query := fmt.Sprintf(`query {
		viewer {
			home(id:"%s") {
				%s
			}
		}
	}`, homeId, innerQuery)

	reqBody, err := json.Marshal(queryRequest{Query: query})
	if err != nil {
		return nil, err
	}

	url := "https://api.tibber.com/v1-beta/gql"
	req, _ := http.NewRequestWithContext(ctx, "POST", url, bytes.NewBuffer(reqBody))
	req.Header.Set("Authorization", fmt.Sprintf("Bearer %s", apiToken))
	req.Header.Set("Accept", "application/json")
	req.Header.Set("Content-Type", "application/json")

	client := http.Client{Timeout: 10 * time.Second}
	res, err := client.Do(req)
	if err != nil {
		return nil, err
	}
	if res.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("got status %s", res.Status)
	}

	defer res.Body.Close()

	bytes, err := io.ReadAll(res.Body)
	if err != nil {
		return nil, err
	}

	resBody := new(queryResponse[T])
	if err = json.Unmarshal(bytes, resBody); err != nil {
		return nil, err
	}

	if resBody.Errors != nil {
		messages := make([]string, len(resBody.Errors))
		for i, err := range resBody.Errors {
			messages[i] = err.Message
		}
		return nil, fmt.Errorf("graphql error: %s", strings.Join(messages, "; "))
	}

	return resBody, nil
}

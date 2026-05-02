package firefly

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"
)

type Account struct {
	ID         string `json:"id"`
	Attributes struct {
		Name           string `json:"name"`
		Type           string `json:"type"`
		CurrentBalance string `json:"current_balance"`
		CurrencyCode   string `json:"currency_code"`
		Active         bool   `json:"active"`
	} `json:"attributes"`
}

type Meta struct {
	Pagination struct {
		Total       int `json:"total"`
		Count       int `json:"count"`
		PerPage     int `json:"per_page"`
		CurrentPage int `json:"current_page"`
		TotalPages  int `json:"total_pages"`
	} `json:"pagination"`
}

type AccountResponse struct {
	Data []Account `json:"data"`
	Meta Meta      `json:"meta"`
}

func GetAccounts(baseURL, token string) ([]Account, error) {
	if baseURL == "" || token == "" {
		return nil, fmt.Errorf("Firefly URL or token not configured")
	}

	// Ensure baseURL ends with /api/v1
	baseURL = strings.TrimSuffix(baseURL, "/")
	if !strings.HasSuffix(baseURL, "/api/v1") {
		baseURL = baseURL + "/api/v1"
	}

	var allAccounts []Account
	page := 1

	for {
		url := fmt.Sprintf("%s/accounts?page=%d", baseURL, page)
		req, err := http.NewRequest("GET", url, nil)
		if err != nil {
			return nil, err
		}

		req.Header.Set("Authorization", "Bearer "+token)
		req.Header.Set("Accept", "application/json")

		resp, err := http.DefaultClient.Do(req)
		if err != nil {
			return nil, err
		}
		defer resp.Body.Close()

		if resp.StatusCode != http.StatusOK {
			body, _ := io.ReadAll(resp.Body)
			return nil, fmt.Errorf("Firefly API error: %d %s", resp.StatusCode, string(body))
		}

		var accountResp AccountResponse
		if err := json.NewDecoder(resp.Body).Decode(&accountResp); err != nil {
			return nil, err
		}

		allAccounts = append(allAccounts, accountResp.Data...)

		if accountResp.Meta.Pagination.TotalPages == 0 || accountResp.Meta.Pagination.CurrentPage >= accountResp.Meta.Pagination.TotalPages {
			break
		}
		page++
	}

	return allAccounts, nil
}

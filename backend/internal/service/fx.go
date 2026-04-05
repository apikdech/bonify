package service

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"time"

	"github.com/receipt-manager/backend/internal/repository"
)

// FXService provides foreign exchange rate operations
type FXService struct {
	fxRepo *repository.FXRepo
	client *http.Client
}

// NewFXService creates a new FX service
func NewFXService(fxRepo *repository.FXRepo) *FXService {
	return &FXService{
		fxRepo: fxRepo,
		client: &http.Client{Timeout: 30 * time.Second},
	}
}

// FrankfurterResponse represents the API response from Frankfurter
type FrankfurterResponse struct {
	Amount float64            `json:"amount"`
	Base   string             `json:"base"`
	Date   string             `json:"date"`
	Rates  map[string]float64 `json:"rates"`
}

// FetchFromFrankfurter fetches FX rates from Frankfurter API for a base currency
func (s *FXService) FetchFromFrankfurter(ctx context.Context, base string) error {
	url := fmt.Sprintf("https://api.frankfurter.app/latest?from=%s", base)

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
	if err != nil {
		return fmt.Errorf("failed to create request: %w", err)
	}

	resp, err := s.client.Do(req)
	if err != nil {
		return fmt.Errorf("failed to fetch FX rates: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("Frankfurter API returned status %d", resp.StatusCode)
	}

	var result FrankfurterResponse
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return fmt.Errorf("failed to decode FX rates response: %w", err)
	}

	// Save all rates to the database
	for target, rate := range result.Rates {
		if err := s.fxRepo.SaveRate(ctx, result.Base, target, rate); err != nil {
			return fmt.Errorf("failed to save rate for %s/%s: %w", result.Base, target, err)
		}
	}

	return nil
}

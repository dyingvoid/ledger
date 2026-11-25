package http

import (
	"context"
	"encoding/json"
	"log/slog"
	nethttp "net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/dyingvoid/ledger/services/bff/internal/domain"
)

type fakeCreateCurrency struct{}

func (*fakeCreateCurrency) CreateCurrency(
	_ context.Context,
	params domain.CreateCurrencyParams,
) (domain.Currency, error) {
	return domain.NewCurrency(params.Code, params.Name, params.Ledger, params.AssetScale)
}

func TestHealth(t *testing.T) {
	t.Parallel()

	handler := NewHandler(&fakeCreateCurrency{}, slog.New(slog.DiscardHandler))
	request := httptest.NewRequestWithContext(context.Background(), nethttp.MethodGet, "/healthz", nil)
	response := httptest.NewRecorder()

	handler.Routes().ServeHTTP(response, request)

	if response.Code != nethttp.StatusOK {
		t.Fatalf("status = %d, want %d", response.Code, nethttp.StatusOK)
	}
}

func TestCreateCurrency(t *testing.T) {
	t.Parallel()

	handler := NewHandler(&fakeCreateCurrency{}, slog.New(slog.DiscardHandler))
	request := httptest.NewRequestWithContext(
		context.Background(),
		nethttp.MethodPost,
		"/v1/currencies",
		strings.NewReader(`{"code":"USD","name":"US Dollar","ledger":1,"asset_scale":2}`),
	)
	response := httptest.NewRecorder()

	handler.Routes().ServeHTTP(response, request)

	if response.Code != nethttp.StatusCreated {
		t.Fatalf("status = %d, want %d", response.Code, nethttp.StatusCreated)
	}

	var currency domain.Currency
	if err := json.NewDecoder(response.Body).Decode(&currency); err != nil {
		t.Fatalf("decoding response: %v", err)
	}
	if currency.Code != "USD" {
		t.Fatalf("currency code = %q, want USD", currency.Code)
	}
}

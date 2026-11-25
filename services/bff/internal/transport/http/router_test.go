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

type fakeAccounts struct {
	operation domain.AccountOperation
}

func (f *fakeAccounts) CreateAccount(
	_ context.Context,
	idempotencyKey string,
	params domain.CreateAccountParams,
) (domain.AccountOperation, error) {
	if _, err := domain.NewCreateAccountParams(
		params.UserID,
		params.Name,
		params.Description,
		params.Type,
		params.Currency,
	); err != nil {
		return domain.AccountOperation{}, err
	}
	return domain.AccountOperation{
		ID:     domain.NewAccountOperationID(idempotencyKey),
		Status: domain.AccountOpeningStatusInProgress,
	}, nil
}

func (f *fakeAccounts) GetAccountOperation(
	_ context.Context,
	operationID string,
) (domain.AccountOperation, error) {
	if f.operation.ID == "" {
		f.operation = domain.AccountOperation{
			ID:     operationID,
			Status: domain.AccountOpeningStatusSuccess,
		}
	}
	return f.operation, nil
}

func TestHealth(t *testing.T) {
	t.Parallel()

	handler := NewHandler(&fakeCreateCurrency{}, &fakeAccounts{}, slog.New(slog.DiscardHandler))
	request := httptest.NewRequestWithContext(context.Background(), nethttp.MethodGet, "/healthz", nil)
	response := httptest.NewRecorder()

	handler.Routes().ServeHTTP(response, request)

	if response.Code != nethttp.StatusOK {
		t.Fatalf("status = %d, want %d", response.Code, nethttp.StatusOK)
	}
}

func TestCreateCurrency(t *testing.T) {
	t.Parallel()

	handler := NewHandler(&fakeCreateCurrency{}, &fakeAccounts{}, slog.New(slog.DiscardHandler))
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

func TestCreateAccount(t *testing.T) {
	t.Parallel()

	handler := NewHandler(&fakeCreateCurrency{}, &fakeAccounts{}, slog.New(slog.DiscardHandler))
	request := httptest.NewRequestWithContext(
		context.Background(),
		nethttp.MethodPost,
		"/v1/accounts",
		strings.NewReader(`{
			"user_id": 42,
			"name": "Main account",
			"description": "Client deposits",
			"type": 2100,
			"currency": "USD"
		}`),
	)
	request.Header.Set("Idempotency-Key", "create-main-account")
	response := httptest.NewRecorder()

	handler.Routes().ServeHTTP(response, request)

	if response.Code != nethttp.StatusAccepted {
		t.Fatalf("status = %d, want %d", response.Code, nethttp.StatusAccepted)
	}

	var operation domain.AccountOperation
	if err := json.NewDecoder(response.Body).Decode(&operation); err != nil {
		t.Fatalf("decoding response: %v", err)
	}
	if operation.Status != domain.AccountOpeningStatusInProgress {
		t.Fatalf("operation status = %q, want %q", operation.Status, domain.AccountOpeningStatusInProgress)
	}
}

func TestCreateAccountRequiresIdempotencyKey(t *testing.T) {
	t.Parallel()

	handler := NewHandler(&fakeCreateCurrency{}, &fakeAccounts{}, slog.New(slog.DiscardHandler))
	request := httptest.NewRequestWithContext(
		context.Background(),
		nethttp.MethodPost,
		"/v1/accounts",
		strings.NewReader(`{"user_id":42,"name":"Main account","type":2100,"currency":"USD"}`),
	)
	response := httptest.NewRecorder()

	handler.Routes().ServeHTTP(response, request)

	if response.Code != nethttp.StatusBadRequest {
		t.Fatalf("status = %d, want %d", response.Code, nethttp.StatusBadRequest)
	}
}

func TestGetAccountOperation(t *testing.T) {
	t.Parallel()

	operationID := domain.NewAccountOperationID("create-main-account")
	handler := NewHandler(
		&fakeCreateCurrency{},
		&fakeAccounts{operation: domain.AccountOperation{
			ID:     operationID,
			Status: domain.AccountOpeningStatusSuccess,
		}},
		slog.New(slog.DiscardHandler),
	)
	request := httptest.NewRequestWithContext(
		context.Background(),
		nethttp.MethodGet,
		"/v1/accounts/"+operationID,
		nil,
	)
	response := httptest.NewRecorder()

	handler.Routes().ServeHTTP(response, request)

	if response.Code != nethttp.StatusOK {
		t.Fatalf("status = %d, want %d", response.Code, nethttp.StatusOK)
	}

	var operation domain.AccountOperation
	if err := json.NewDecoder(response.Body).Decode(&operation); err != nil {
		t.Fatalf("decoding response: %v", err)
	}
	if operation.Status != domain.AccountOpeningStatusSuccess {
		t.Fatalf("operation status = %q, want %q", operation.Status, domain.AccountOpeningStatusSuccess)
	}
}

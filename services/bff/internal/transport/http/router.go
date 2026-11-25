package http

import (
	"context"
	"encoding/json"
	"errors"
	"log/slog"
	nethttp "net/http"

	"github.com/dyingvoid/ledger/services/bff/internal/domain"
	webstatic "github.com/dyingvoid/ledger/services/bff/static"
)

type CreateCurrency interface {
	CreateCurrency(ctx context.Context, params domain.CreateCurrencyParams) (domain.Currency, error)
}

type Accounts interface {
	CreateAccount(
		ctx context.Context,
		idempotencyKey string,
		params domain.CreateAccountParams,
	) (domain.AccountOperation, error)
	GetAccountOperation(ctx context.Context, operationID string) (domain.AccountOperation, error)
}

type Handler struct {
	createCurrency CreateCurrency
	accounts       Accounts
	log            *slog.Logger
}

func NewHandler(createCurrency CreateCurrency, accounts Accounts, log *slog.Logger) *Handler {
	return &Handler{createCurrency: createCurrency, accounts: accounts, log: log}
}

func (h *Handler) Routes() nethttp.Handler {
	mux := nethttp.NewServeMux()
	mux.HandleFunc("GET /{$}", func(w nethttp.ResponseWriter, r *nethttp.Request) {
		nethttp.Redirect(w, r, "/static/", nethttp.StatusFound)
	})
	mux.Handle(
		"GET /static/",
		nethttp.StripPrefix("/static/", nethttp.FileServerFS(webstatic.Files())),
	)
	mux.HandleFunc("GET /healthz", h.health)
	mux.HandleFunc("POST /v1/currencies", h.createCurrencyHandler)
	mux.HandleFunc("POST /v1/accounts", h.createAccountHandler)
	mux.HandleFunc("GET /v1/accounts/{operation_id}", h.getAccountOperationHandler)
	return mux
}

func (h *Handler) health(w nethttp.ResponseWriter, _ *nethttp.Request) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(nethttp.StatusOK)
	_ = json.NewEncoder(w).Encode(map[string]string{"status": "ok"})
}

func (h *Handler) createCurrencyHandler(w nethttp.ResponseWriter, r *nethttp.Request) {
	var request struct {
		Code       string `json:"code"`
		Name       string `json:"name"`
		Ledger     uint32 `json:"ledger"`
		AssetScale int32  `json:"asset_scale"`
	}

	r.Body = nethttp.MaxBytesReader(w, r.Body, 1<<20)
	decoder := json.NewDecoder(r.Body)
	decoder.DisallowUnknownFields()
	if err := decoder.Decode(&request); err != nil {
		nethttp.Error(w, "invalid request body", nethttp.StatusBadRequest)
		return
	}

	currency, err := h.createCurrency.CreateCurrency(r.Context(), domain.CreateCurrencyParams{
		Code:       request.Code,
		Name:       request.Name,
		Ledger:     request.Ledger,
		AssetScale: request.AssetScale,
	})
	if err != nil {
		if isValidationError(err) {
			nethttp.Error(w, err.Error(), nethttp.StatusBadRequest)
			return
		}
		h.log.ErrorContext(r.Context(), "creating currency", "error", err)
		nethttp.Error(w, "creating currency", nethttp.StatusBadGateway)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(nethttp.StatusCreated)
	_ = json.NewEncoder(w).Encode(currency)
}

func (h *Handler) createAccountHandler(w nethttp.ResponseWriter, r *nethttp.Request) {
	idempotencyKey := r.Header.Get("Idempotency-Key")
	if idempotencyKey == "" {
		nethttp.Error(w, domain.ErrIdempotencyKeyRequired.Error(), nethttp.StatusBadRequest)
		return
	}

	var request struct {
		UserID      int64  `json:"user_id"`
		Name        string `json:"name"`
		Description string `json:"description"`
		Type        int32  `json:"type"`
		Currency    string `json:"currency"`
	}

	r.Body = nethttp.MaxBytesReader(w, r.Body, 1<<20)
	decoder := json.NewDecoder(r.Body)
	decoder.DisallowUnknownFields()
	if err := decoder.Decode(&request); err != nil {
		nethttp.Error(w, "invalid request body", nethttp.StatusBadRequest)
		return
	}

	operation, err := h.accounts.CreateAccount(r.Context(), idempotencyKey, domain.CreateAccountParams{
		UserID:      request.UserID,
		Name:        request.Name,
		Description: request.Description,
		Type:        request.Type,
		Currency:    request.Currency,
	})
	if err != nil {
		if isAccountValidationError(err) {
			nethttp.Error(w, err.Error(), nethttp.StatusBadRequest)
			return
		}
		h.log.ErrorContext(r.Context(), "creating account", "error", err)
		nethttp.Error(w, "creating account", nethttp.StatusBadGateway)
		return
	}

	statusCode := nethttp.StatusOK
	if operation.Status == domain.AccountOpeningStatusInProgress {
		statusCode = nethttp.StatusAccepted
	}
	writeAccountOperation(w, operation, statusCode)
}

func (h *Handler) getAccountOperationHandler(w nethttp.ResponseWriter, r *nethttp.Request) {
	operation, err := h.accounts.GetAccountOperation(r.Context(), r.PathValue("operation_id"))
	if err != nil {
		switch {
		case errors.Is(err, domain.ErrAccountOperationIDInvalid):
			nethttp.Error(w, err.Error(), nethttp.StatusBadRequest)
		case errors.Is(err, domain.ErrAccountOperationNotFound):
			nethttp.Error(w, err.Error(), nethttp.StatusNotFound)
		default:
			h.log.ErrorContext(r.Context(), "getting account operation", "error", err)
			nethttp.Error(w, "getting account operation", nethttp.StatusBadGateway)
		}
		return
	}
	writeAccountOperation(w, operation, nethttp.StatusOK)
}

func writeAccountOperation(
	w nethttp.ResponseWriter,
	operation domain.AccountOperation,
	statusCode int,
) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(statusCode)
	_ = json.NewEncoder(w).Encode(operation)
}

func isValidationError(err error) bool {
	return errors.Is(err, domain.ErrCurrencyCodeRequired) ||
		errors.Is(err, domain.ErrCurrencyNameRequired) ||
		errors.Is(err, domain.ErrCurrencyLedgerInvalid) ||
		errors.Is(err, domain.ErrCurrencyAssetScaleInvalid)
}

func isAccountValidationError(err error) bool {
	return errors.Is(err, domain.ErrIdempotencyKeyRequired) ||
		errors.Is(err, domain.ErrAccountUserIDInvalid) ||
		errors.Is(err, domain.ErrAccountNameRequired) ||
		errors.Is(err, domain.ErrAccountTypeInvalid) ||
		errors.Is(err, domain.ErrAccountCurrencyRequired)
}

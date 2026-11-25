package http

import (
	"context"
	"encoding/json"
	"errors"
	"log/slog"
	nethttp "net/http"

	"github.com/dyingvoid/ledger/services/bff/internal/domain"
)

type CreateCurrency interface {
	CreateCurrency(ctx context.Context, params domain.CreateCurrencyParams) (domain.Currency, error)
}

type Handler struct {
	createCurrency CreateCurrency
	log            *slog.Logger
}

func NewHandler(createCurrency CreateCurrency, log *slog.Logger) *Handler {
	return &Handler{createCurrency: createCurrency, log: log}
}

func (h *Handler) Routes() nethttp.Handler {
	mux := nethttp.NewServeMux()
	mux.HandleFunc("GET /healthz", h.health)
	mux.HandleFunc("POST /v1/currencies", h.createCurrencyHandler)
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

func isValidationError(err error) bool {
	return errors.Is(err, domain.ErrCurrencyCodeRequired) ||
		errors.Is(err, domain.ErrCurrencyNameRequired) ||
		errors.Is(err, domain.ErrCurrencyLedgerInvalid) ||
		errors.Is(err, domain.ErrCurrencyAssetScaleInvalid)
}

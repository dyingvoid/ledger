package deposits

import (
	"context"
	"errors"
	"fmt"
	"strings"

	"github.com/dyingvoid/ledger/internal/domain"
	domain_errors "github.com/dyingvoid/ledger/internal/domain/errors"
	"github.com/dyingvoid/ledger/internal/logger"
)

// TODO: to errors.go
var ErrLedgerUnavailable = errors.New("ledger unavailable")

type AccountRepository interface {
	Get(ctx context.Context, id domain.ID) (domain.Account, error)
	GetBankCash(ctx context.Context, currency string) (domain.Account, error)
}

type CurrencyRepository interface {
	Get(ctx context.Context, code string) (domain.CurrencyInfo, error)
}

type LedgerRepository interface {
	Create(ctx context.Context, deposit domain.Deposit) (domain.Deposit, error)
}

type Service struct {
	accounts   AccountRepository
	currencies CurrencyRepository
	ledger     LedgerRepository
}

func NewService(
	accounts AccountRepository,
	currencies CurrencyRepository,
	ledger LedgerRepository,
) *Service {
	return &Service{accounts: accounts, currencies: currencies, ledger: ledger}
}

type DepositParams struct {
	ID          domain.ID
	AccountID   domain.ID
	AmountMinor int64
	Currency    string
}

func (s *Service) Deposit(ctx context.Context, params DepositParams) (domain.Deposit, error) {
	if params.ID.IsZero() {
		return domain.Deposit{}, domain_errors.ErrDepositIDRequired
	}
	if params.AmountMinor <= 0 {
		return domain.Deposit{}, domain_errors.ErrDepositAmountInvalid
	}
	if strings.TrimSpace(params.Currency) == "" {
		return domain.Deposit{}, domain_errors.ErrDepositCurrencyRequired
	}
	currencyCode := strings.TrimSpace(params.Currency)

	// TODO: errgroup and 3 fetches concurrently(client, bank, currency)
	clientAccount, err := s.accounts.Get(ctx, params.AccountID)
	if err != nil {
		return domain.Deposit{}, fmt.Errorf("getting client account: %w", err)
	}

	currency, err := s.currencies.Get(ctx, currencyCode)
	if err != nil {
		return domain.Deposit{}, fmt.Errorf("getting currency: %w", err)
	}

	bankAccount, err := s.accounts.GetBankCash(ctx, currency.Code)
	if err != nil {
		return domain.Deposit{}, fmt.Errorf("getting bank cash account: %w", err)
	}

	deposit, err := domain.NewDeposit(
		params.ID,
		params.AmountMinor,
		currency,
		clientAccount,
		bankAccount,
	)
	if err != nil {
		return domain.Deposit{}, fmt.Errorf("validating deposit: %w", err)
	}

	created, err := s.ledger.Create(ctx, deposit)
	if err != nil {
		return domain.Deposit{}, fmt.Errorf("creating deposit: %w", err)
	}

	// TODO: private info, only log deposit created, transfer-id
	logger.FromContext(ctx).InfoContext(
		ctx,
		"deposit created",
		"transfer_id", created.ID,
		"account_id", created.ClientAccount.ID,
		"bank_account_id", created.BankAccount.ID,
		"amount_minor", created.AmountMinor,
		"currency", created.Currency.Code,
	)

	return created, nil
}

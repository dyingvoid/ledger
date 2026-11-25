package accounts

import (
	"context"
	"errors"
	"fmt"

	"github.com/dyingvoid/ledger/services/accounting/internal/domain"
	domain_errors "github.com/dyingvoid/ledger/services/accounting/internal/domain/errors"
	"github.com/dyingvoid/ledger/services/accounting/internal/logger"
)

func (s *Service) OpenLedgerAccount(ctx context.Context, id domain.ID) (domain.AccountLedger, error) {
	account, err := s.infos.Get(ctx, id)
	if err != nil {
		return domain.AccountLedger{}, fmt.Errorf("reading account info: %w", err)
	}

	existing, err := s.ledger.Get(ctx, id)
	switch {
	case err == nil:
		if existing.UserID != account.UserID {
			return domain.AccountLedger{}, fmt.Errorf(
				"%w: account %s belongs to user %d, requested by user %d",
				domain_errors.ErrAccountOwnedByAnotherUser,
				id,
				existing.UserID,
				account.UserID,
			)
		}

		logger.FromContext(ctx).InfoContext(
			ctx,
			"ledger account already open",
			"account_id", account.ID,
			"user_id", account.UserID,
		)

		return existing, nil
	case !errors.Is(err, ErrAccountNotFound):
		return domain.AccountLedger{}, fmt.Errorf("checking ledger account: %w", err)
	}

	opened, err := s.ledger.Create(ctx, account)
	if err != nil {
		return domain.AccountLedger{}, fmt.Errorf("open ledger account: %w", err)
	}

	logger.FromContext(ctx).InfoContext(
		ctx,
		"ledger account opened",
		"account_id", opened.ID,
		"user_id", opened.UserID,
	)

	return opened, nil
}

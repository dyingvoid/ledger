package accounts

import (
	"context"
	"fmt"

	"github.com/dyingvoid/ledger/internal/domain"
	"github.com/dyingvoid/ledger/internal/logger"
)

func (s *Service) OpenLedgerAccount(ctx context.Context, id domain.ID) (domain.Account, error) {
	account, err := s.infos.Get(ctx, id)
	if err != nil {
		return domain.Account{}, fmt.Errorf("reading account info: %w", err)
	}

	opened, err := s.ledger.Create(ctx, account)
	if err != nil {
		return domain.Account{}, fmt.Errorf("open ledger account: %w", err)
	}

	logger.FromContext(ctx).InfoContext(
		ctx,
		"ledger account opened",
		"account_id", opened.ID,
		"user_id", opened.UserID,
	)

	return opened, nil
}

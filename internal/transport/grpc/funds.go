package grpc

import (
	"context"
	"crypto/sha256"
	"encoding/hex"

	"github.com/dyingvoid/ledger/internal/core/deposits"
	"github.com/dyingvoid/ledger/internal/core/idempotency"
	"github.com/dyingvoid/ledger/internal/domain"
	fundsv1 "github.com/dyingvoid/ledger/pkg/proto/ledger/funds/v1"
)

type fundsService struct {
	fundsv1.UnimplementedFundsServiceServer
	deposits           *deposits.Service
	idempotencyService *idempotency.Service
}

func newFundsService(
	depositsService *deposits.Service,
	idempotencyService *idempotency.Service,
) *fundsService {
	return &fundsService{
		deposits:           depositsService,
		idempotencyService: idempotencyService,
	}
}

func (s *fundsService) Deposit(
	ctx context.Context,
	req *fundsv1.DepositRequest,
) (*fundsv1.DepositResponse, error) {
	identity, err := parseRequestIdentity(ctx)
	if err != nil {
		return nil, err
	}

	key := identity.scopedKey("funds.deposit", identity.externalID.String())
	transferID, err := depositTransferID(key)
	if err != nil {
		return nil, toStatus(err)
	}

	deposit, replayed, err := s.idempotencyService.Execute(ctx, key, func(ctx context.Context) (domain.Deposit, error) {
		return s.deposits.AddDeposit(ctx, deposits.AddDepositParams{
			AccountID:         req.GetAccountId(),
			ExternalAccountID: identity.externalID,
			TransferID:        transferID,
			AmountMinor:       req.GetAmountMinor(),
			Currency:          domain.Currency(req.GetCurrency()),
		})
	})
	if err != nil {
		return nil, toStatus(err)
	}

	if err := setReplayHeader(ctx, replayed); err != nil {
		return nil, err
	}

	response := &fundsv1.DepositResponse{
		JournalEntryId: deposit.JournalEntryID,
		Replayed:       replayed,
	}
	if deposit.JournalEntryID == 0 && !deposit.TransferID.IsZero() {
		response.TransferId = deposit.TransferID.String()
	}

	return response, nil
}

func depositTransferID(idempotencyKey string) (domain.ExternalID, error) {
	digest := sha256.Sum256([]byte("ledger.deposit.v1:" + idempotencyKey))
	return domain.ParseExternalID("0x" + hex.EncodeToString(digest[:16]))
}

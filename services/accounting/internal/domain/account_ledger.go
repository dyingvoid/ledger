package domain

import domain_errors "github.com/dyingvoid/ledger/services/accounting/internal/domain/errors"

type AccountLedger struct {
	ID     ID          `json:"ledger_id"`
	UserID int64       `json:"user_id"`
	Ledger uint32      `json:"ledger"`
	Type   AccountType `json:"type"`
}

func NewAccountLedger(
	id ID,
	userID int64,
	ledger uint32,
	accountType AccountType,
) (AccountLedger, error) {
	account := AccountLedger{
		ID:     id,
		UserID: userID,
		Ledger: ledger,
		Type:   accountType,
	}
	if err := account.Validate(); err != nil {
		return AccountLedger{}, err
	}
	return account, nil
}

func (a AccountLedger) NormalBalance() (BalanceSide, error) {
	return a.Type.NormalBalance()
}

func (a AccountLedger) Validate() error {
	if a.ID.IsZero() {
		return domain_errors.ErrAccountIDRequired
	}
	if a.UserID <= 0 {
		return domain_errors.ErrAccountUserRequired
	}
	if a.Ledger == 0 {
		return domain_errors.ErrAccountLedgerRequired
	}
	return a.Type.Validate()
}

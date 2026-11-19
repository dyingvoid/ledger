package domain

import (
	"fmt"

	domain_errors "github.com/dyingvoid/ledger/internal/domain/errors"
)

type Deposit struct {
	ID            ID           `json:"id"`
	AmountMinor   int64        `json:"amount_minor"`
	Currency      CurrencyInfo `json:"currency"`
	ClientAccount Account      `json:"client_account"`
	BankAccount   Account      `json:"bank_account"`
}

func NewDeposit(
	id ID,
	amountMinor int64,
	currency CurrencyInfo,
	clientAccount Account,
	bankAccount Account,
) (Deposit, error) {
	deposit := Deposit{
		ID:            id,
		AmountMinor:   amountMinor,
		Currency:      currency,
		ClientAccount: clientAccount,
		BankAccount:   bankAccount,
	}
	if err := deposit.Validate(); err != nil {
		return Deposit{}, err
	}
	return deposit, nil
}

func (d Deposit) Validate() error {
	if d.ID.IsZero() {
		return domain_errors.ErrDepositIDRequired
	}
	if d.AmountMinor <= 0 {
		return domain_errors.ErrDepositAmountInvalid
	}
	if err := d.Currency.Validate(); err != nil {
		return fmt.Errorf("deposit currency: %w", err)
	}
	if d.ClientAccount.ID.IsZero() {
		return domain_errors.ErrDepositClientAccountRequired
	}
	if d.BankAccount.ID.IsZero() {
		return domain_errors.ErrDepositBankAccountRequired
	}
	if d.ClientAccount.ID == d.BankAccount.ID {
		return domain_errors.ErrDepositAccountsIdentical
	}
	if d.ClientAccount.DeletedAt != nil {
		return fmt.Errorf("%w: client account %s", domain_errors.ErrDepositAccountDeleted, d.ClientAccount.ID)
	}
	if d.BankAccount.DeletedAt != nil {
		return fmt.Errorf("%w: bank account %s", domain_errors.ErrDepositAccountDeleted, d.BankAccount.ID)
	}
	if !d.ClientAccount.Type.SupportsDeposits() {
		return fmt.Errorf(
			"%w: %d",
			domain_errors.ErrDepositClientAccountTypeUnsupported,
			d.ClientAccount.Type,
		)
	}
	if !d.BankAccount.Type.CanFundDeposits() {
		return fmt.Errorf(
			"%w: %d",
			domain_errors.ErrDepositBankAccountTypeUnsupported,
			d.BankAccount.Type,
		)
	}
	if d.ClientAccount.Currency != d.Currency.Code {
		return fmt.Errorf(
			"%w: client account %q, deposit %q",
			domain_errors.ErrDepositClientCurrencyMismatch,
			d.ClientAccount.Currency,
			d.Currency.Code,
		)
	}
	if d.BankAccount.Currency != d.Currency.Code {
		return fmt.Errorf(
			"%w: bank account %q, deposit %q",
			domain_errors.ErrDepositBankCurrencyMismatch,
			d.BankAccount.Currency,
			d.Currency.Code,
		)
	}
	return nil
}

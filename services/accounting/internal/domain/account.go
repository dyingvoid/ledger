package domain

type Account struct {
	Info   *AccountInfo   `json:"info"`
	Ledger *AccountLedger `json:"ledger"`
}

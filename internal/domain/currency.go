package domain

type Currency string

// CurrencyInfo maps a currency to its TigerBeetle ledger and asset scale.
type CurrencyInfo struct {
	Code       Currency
	Name       string
	Ledger     uint32
	AssetScale int16
}

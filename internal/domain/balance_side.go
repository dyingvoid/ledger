package domain

type BalanceSide uint8

const (
	Debit BalanceSide = iota + 1
	Credit
)

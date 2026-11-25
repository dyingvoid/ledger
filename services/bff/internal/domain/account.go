package domain

import (
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"strings"
	"time"
)

var (
	ErrIdempotencyKeyRequired    = errors.New("idempotency key is required")
	ErrAccountUserIDInvalid      = errors.New("account user id must be positive")
	ErrAccountNameRequired       = errors.New("account name is required")
	ErrAccountTypeInvalid        = errors.New("account type must be between 1000 and 5999")
	ErrAccountCurrencyRequired   = errors.New("account currency is required")
	ErrAccountOperationNotFound  = errors.New("account operation not found")
	ErrAccountingIDEmpty         = errors.New("accounting id is empty")
	ErrAccountOperationIDInvalid = errors.New("account operation id is invalid")
)

type AccountOpeningStatus string

const (
	AccountOpeningStatusUnknown    AccountOpeningStatus = ""
	AccountOpeningStatusInProgress AccountOpeningStatus = "in_progress"
	AccountOpeningStatusSuccess    AccountOpeningStatus = "success"
	AccountOpeningStatusFailure    AccountOpeningStatus = "failure"
)

type CreateAccountParams struct {
	UserID      int64
	Name        string
	Description string
	Type        int32
	Currency    string
}

func NewCreateAccountParams(
	userID int64,
	name string,
	description string,
	accountType int32,
	currency string,
) (CreateAccountParams, error) {
	params := CreateAccountParams{
		UserID:      userID,
		Name:        strings.TrimSpace(name),
		Description: strings.TrimSpace(description),
		Type:        accountType,
		Currency:    strings.TrimSpace(currency),
	}
	if params.UserID <= 0 {
		return CreateAccountParams{}, ErrAccountUserIDInvalid
	}
	if params.Name == "" {
		return CreateAccountParams{}, ErrAccountNameRequired
	}
	if params.Type < 1000 || params.Type > 5999 {
		return CreateAccountParams{}, ErrAccountTypeInvalid
	}
	if params.Currency == "" {
		return CreateAccountParams{}, ErrAccountCurrencyRequired
	}
	return params, nil
}

type AccountOpeningInput struct {
	AccountingID                   string
	UserID                         int64
	Name                           string
	Description                    string
	Type                           int32
	Currency                       string
	ActivityStartToCloseTimeout    time.Duration
	ActivityScheduleToCloseTimeout time.Duration
	ActivityMaximumAttempts        int32
}

type AccountOperation struct {
	ID     string               `json:"operation_id"`
	Status AccountOpeningStatus `json:"status"`
}

func NewAccountOperationID(idempotencyKey string) string {
	hash := sha256.Sum256([]byte("account-opening\x00" + idempotencyKey))
	return hex.EncodeToString(hash[:])
}

func IsAccountOperationID(value string) bool {
	if len(value) != sha256.Size*2 {
		return false
	}
	_, err := hex.DecodeString(value)
	return err == nil
}

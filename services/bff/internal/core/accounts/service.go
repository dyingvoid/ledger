package accounts

import (
	"context"
	"errors"
	"fmt"
	"strings"

	"github.com/dyingvoid/ledger/services/bff/internal/domain"
)

type Workflow interface {
	StartAccountOpening(
		ctx context.Context,
		operationID string,
		input domain.AccountOpeningInput,
	) (domain.AccountOpeningStatus, error)
	GetAccountOpeningStatus(
		ctx context.Context,
		operationID string,
	) (domain.AccountOpeningStatus, error)
}

type IDGenerator interface {
	NewID() string
}

type Service struct {
	workflow Workflow
	ids      IDGenerator
}

func NewService(workflow Workflow, ids IDGenerator) *Service {
	return &Service{workflow: workflow, ids: ids}
}

func (s *Service) CreateAccount(
	ctx context.Context,
	idempotencyKey string,
	params domain.CreateAccountParams,
) (domain.AccountOperation, error) {
	if strings.TrimSpace(idempotencyKey) == "" {
		return domain.AccountOperation{}, domain.ErrIdempotencyKeyRequired
	}

	account, err := domain.NewCreateAccountParams(
		params.UserID,
		params.Name,
		params.Description,
		params.Type,
		params.Currency,
	)
	if err != nil {
		return domain.AccountOperation{}, fmt.Errorf("validating account: %w", err)
	}

	accountingID := s.ids.NewID()
	if accountingID == "" {
		return domain.AccountOperation{}, domain.ErrAccountingIDEmpty
	}

	operationID := domain.NewAccountOperationID(idempotencyKey)
	status, err := s.workflow.StartAccountOpening(ctx, operationID, domain.AccountOpeningInput{
		AccountingID: accountingID,
		UserID:       account.UserID,
		Name:         account.Name,
		Description:  account.Description,
		Type:         account.Type,
		Currency:     account.Currency,
	})
	if err != nil {
		return domain.AccountOperation{}, fmt.Errorf("starting account opening workflow: %w", err)
	}
	return domain.AccountOperation{ID: operationID, Status: status}, nil
}

func (s *Service) GetAccountOperation(
	ctx context.Context,
	operationID string,
) (domain.AccountOperation, error) {
	if !domain.IsAccountOperationID(operationID) {
		return domain.AccountOperation{}, domain.ErrAccountOperationIDInvalid
	}

	status, err := s.workflow.GetAccountOpeningStatus(ctx, operationID)
	if err != nil {
		if errors.Is(err, domain.ErrAccountOperationNotFound) {
			return domain.AccountOperation{}, err
		}
		return domain.AccountOperation{}, fmt.Errorf("getting account opening status: %w", err)
	}
	return domain.AccountOperation{ID: operationID, Status: status}, nil
}

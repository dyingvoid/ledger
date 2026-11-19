package grpc

import (
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"

	"github.com/dyingvoid/ledger/internal/domain"
)

func parseID(value string) (domain.ID, error) {
	id, err := domain.ParseID(value)
	if err != nil {
		return domain.ID{}, status.Error(codes.InvalidArgument, err.Error())
	}
	return id, nil
}

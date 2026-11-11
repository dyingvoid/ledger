package grpc

import (
	"context"
	"strings"

	grpcpkg "google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/metadata"
	"google.golang.org/grpc/status"

	"github.com/dyingvoid/ledger/internal/domain"
)

const (
	idempotencyKeyHeader      = "x-idempotency-key"
	idempotencyReplayedHeader = "x-idempotent-replayed"
)

type requestIdentity struct {
	key        string
	externalID domain.ExternalID
}

func parseRequestIdentity(ctx context.Context) (requestIdentity, error) {
	md, ok := metadata.FromIncomingContext(ctx)
	if !ok {
		return requestIdentity{}, status.Errorf(codes.InvalidArgument, "%s metadata is required", idempotencyKeyHeader)
	}

	values := md.Get(idempotencyKeyHeader)
	if len(values) == 0 || strings.TrimSpace(values[0]) == "" {
		return requestIdentity{}, status.Errorf(codes.InvalidArgument, "%s metadata is required", idempotencyKeyHeader)
	}

	key := strings.TrimSpace(values[0])
	externalValue := key
	if index := strings.LastIndex(externalValue, ":"); index >= 0 {
		externalValue = externalValue[index+1:]
	}

	externalID, err := domain.ParseExternalID(externalValue)
	if err != nil {
		return requestIdentity{}, status.Error(codes.InvalidArgument, err.Error())
	}

	return requestIdentity{key: key, externalID: externalID}, nil
}

func (i requestIdentity) scopedKey(operation string, scope string) string {
	return operation + ":" + scope + ":" + i.key
}

func setReplayHeader(ctx context.Context, replayed bool) error {
	if !replayed {
		return nil
	}
	if err := grpcpkg.SetHeader(ctx, metadata.Pairs(idempotencyReplayedHeader, "true")); err != nil {
		return status.Errorf(codes.Internal, "setting replay header: %v", err)
	}
	return nil
}

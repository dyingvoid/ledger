package grpc

import (
	"context"
	"strconv"
	"strings"

	grpcpkg "google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/metadata"
	"google.golang.org/grpc/status"
)

const (
	idempotencyKeyHeader      = "x-idempotency-key"
	idempotencyReplayedHeader = "x-idempotent-replayed"
)

func idempotencyKey(ctx context.Context, operation string, scope int64) (string, error) {
	md, ok := metadata.FromIncomingContext(ctx)
	if !ok {
		return "", status.Errorf(codes.InvalidArgument, "%s metadata is required", idempotencyKeyHeader)
	}

	values := md.Get(idempotencyKeyHeader)
	if len(values) == 0 || strings.TrimSpace(values[0]) == "" {
		return "", status.Errorf(codes.InvalidArgument, "%s metadata is required", idempotencyKeyHeader)
	}

	return operation + ":" + strconv.FormatInt(scope, 10) + ":" + strings.TrimSpace(values[0]), nil
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

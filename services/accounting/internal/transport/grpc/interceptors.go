package grpc

import (
	"context"
	"log/slog"
	"runtime/debug"

	grpcpkg "google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"

	"github.com/dyingvoid/ledger/services/accounting/internal/logger"
)

const grpcMethodKey = "grpc_method"

func loggerInterceptor(log *slog.Logger) grpcpkg.UnaryServerInterceptor {
	return func(ctx context.Context, req any, info *grpcpkg.UnaryServerInfo, handler grpcpkg.UnaryHandler) (any, error) {
		ctx = logger.WithContext(ctx, log.With(grpcMethodKey, info.FullMethod))
		return handler(ctx, req)
	}
}

func statusErrorInterceptor() grpcpkg.UnaryServerInterceptor {
	return func(ctx context.Context, req any, _ *grpcpkg.UnaryServerInfo, handler grpcpkg.UnaryHandler) (any, error) {
		response, err := handler(ctx, req)
		if err == nil {
			return response, nil
		}
		if _, ok := status.FromError(err); ok {
			return response, err
		}
		return response, toStatus(err)
	}
}

func recoverInterceptor() grpcpkg.UnaryServerInterceptor {
	return func(ctx context.Context, req any, info *grpcpkg.UnaryServerInfo, handler grpcpkg.UnaryHandler) (response any, err error) {
		defer func() {
			recovered := recover()
			if recovered == nil {
				return
			}

			logger.FromContext(ctx).ErrorContext(
				ctx,
				"grpc handler panic",
				grpcMethodKey, info.FullMethod,
				"panic", recovered,
				"stack", string(debug.Stack()),
			)

			response = nil
			err = status.Error(codes.Internal, "internal server error")
		}()

		return handler(ctx, req)
	}
}

func traceInterceptor() grpcpkg.UnaryServerInterceptor {
	return func(ctx context.Context, req any, info *grpcpkg.UnaryServerInfo, handler grpcpkg.UnaryHandler) (any, error) {
		log := logger.FromContext(ctx)
		log.DebugContext(ctx, "incoming grpc request", grpcMethodKey, info.FullMethod)

		response, err := handler(ctx, req)

		log.DebugContext(
			ctx,
			"grpc response",
			grpcMethodKey, info.FullMethod,
			"grpc_code", status.Code(err).String(),
		)

		return response, err
	}
}

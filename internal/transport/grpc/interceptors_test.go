package grpc

import (
	"bytes"
	"context"
	"log/slog"
	"strings"
	"testing"

	grpcpkg "google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"

	"github.com/dyingvoid/ledger/internal/logger"
)

func TestLoggerInterceptor(t *testing.T) {
	t.Parallel()

	base := slog.New(slog.DiscardHandler)
	interceptor := loggerInterceptor(base)

	_, err := interceptor(
		context.Background(),
		nil,
		&grpcpkg.UnaryServerInfo{FullMethod: "/test.Service/Method"},
		func(ctx context.Context, _ any) (any, error) {
			if logger.FromContext(ctx) == base {
				t.Error("handler context logger is the base logger, want request logger")
			}
			return nil, nil
		},
	)
	if err != nil {
		t.Fatalf("interceptor error = %v", err)
	}
}

func TestRecoverInterceptor(t *testing.T) {
	t.Parallel()

	var output bytes.Buffer
	base := slog.New(slog.NewTextHandler(&output, nil))
	ctx := logger.WithContext(context.Background(), base)
	interceptor := recoverInterceptor()

	_, err := interceptor(
		ctx,
		nil,
		&grpcpkg.UnaryServerInfo{FullMethod: "/test.Service/Panic"},
		func(context.Context, any) (any, error) {
			panic("boom")
		},
	)
	if status.Code(err) != codes.Internal {
		t.Fatalf("status code = %v, want %v", status.Code(err), codes.Internal)
	}
	if !strings.Contains(output.String(), "grpc handler panic") {
		t.Fatalf("output %q does not contain panic log", output.String())
	}
}

func TestTraceInterceptor(t *testing.T) {
	t.Parallel()

	var output bytes.Buffer
	base := slog.New(slog.NewTextHandler(&output, &slog.HandlerOptions{Level: slog.LevelDebug}))
	ctx := logger.WithContext(context.Background(), base)
	interceptor := traceInterceptor()

	_, err := interceptor(
		ctx,
		nil,
		&grpcpkg.UnaryServerInfo{FullMethod: "/test.Service/Method"},
		func(context.Context, any) (any, error) {
			return nil, status.Error(codes.InvalidArgument, "invalid")
		},
	)
	if status.Code(err) != codes.InvalidArgument {
		t.Fatalf("status code = %v, want %v", status.Code(err), codes.InvalidArgument)
	}

	logged := output.String()
	if !strings.Contains(logged, "incoming grpc request") {
		t.Fatalf("output %q does not contain request log", logged)
	}
	if !strings.Contains(logged, "grpc_code=InvalidArgument") {
		t.Fatalf("output %q does not contain response code", logged)
	}
}

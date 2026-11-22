package grpc

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"log/slog"
	"strings"
	"testing"

	grpcpkg "google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"

	domain_errors "github.com/dyingvoid/ledger/internal/domain/errors"
	"github.com/dyingvoid/ledger/internal/logger"
)

const testFullMethod = "/test.Service/Method"

func TestLoggerInterceptor(t *testing.T) {
	t.Parallel()

	base := slog.New(slog.DiscardHandler)
	interceptor := loggerInterceptor(base)

	_, err := interceptor(
		context.Background(),
		nil,
		&grpcpkg.UnaryServerInfo{FullMethod: testFullMethod},
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

func TestStatusErrorInterceptor(t *testing.T) {
	t.Parallel()

	t.Run("maps raw error", func(t *testing.T) {
		t.Parallel()

		interceptor := statusErrorInterceptor()

		_, err := interceptor(
			context.Background(),
			nil,
			&grpcpkg.UnaryServerInfo{FullMethod: testFullMethod},
			func(context.Context, any) (any, error) {
				return nil, fmt.Errorf("parsing account id: %w", domain_errors.ErrAccountIDRequired)
			},
		)
		if status.Code(err) != codes.InvalidArgument {
			t.Fatalf("status code = %v, want %v", status.Code(err), codes.InvalidArgument)
		}
	})

	t.Run("preserves grpc status", func(t *testing.T) {
		t.Parallel()

		interceptor := statusErrorInterceptor()
		wantErr := status.Error(codes.PermissionDenied, "permission denied")

		_, err := interceptor(
			context.Background(),
			nil,
			&grpcpkg.UnaryServerInfo{FullMethod: testFullMethod},
			func(context.Context, any) (any, error) {
				return nil, wantErr
			},
		)
		if !errors.Is(err, wantErr) {
			t.Fatalf("error = %v, want original status error", err)
		}
	})

	t.Run("passes nil error", func(t *testing.T) {
		t.Parallel()

		interceptor := statusErrorInterceptor()
		wantResponse := "response"

		response, err := interceptor(
			context.Background(),
			nil,
			&grpcpkg.UnaryServerInfo{FullMethod: testFullMethod},
			func(context.Context, any) (any, error) {
				return wantResponse, nil
			},
		)
		if err != nil {
			t.Fatalf("interceptor error = %v", err)
		}
		if response != wantResponse {
			t.Fatalf("response = %v, want %v", response, wantResponse)
		}
	})
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
		&grpcpkg.UnaryServerInfo{FullMethod: testFullMethod},
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

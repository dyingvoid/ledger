package grpc

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"net"

	grpcpkg "google.golang.org/grpc"
	"google.golang.org/grpc/reflection"

	"github.com/dyingvoid/ledger/internal/core/accounts"
	"github.com/dyingvoid/ledger/internal/core/deposits"
	"github.com/dyingvoid/ledger/internal/core/idempotency"
	accountsv1 "github.com/dyingvoid/ledger/pkg/proto/ledger/accounts/v1"
	fundsv1 "github.com/dyingvoid/ledger/pkg/proto/ledger/funds/v1"
)

type Server struct {
	server   *grpcpkg.Server
	listener net.Listener
}

func NewServer(
	cfg Config,
	log *slog.Logger,
	accountsService *accounts.Service,
	depositsService *deposits.Service,
	idempotencyService *idempotency.Service,
) (*Server, error) {
	listener, err := net.Listen("tcp", cfg.Addr)
	if err != nil {
		return nil, fmt.Errorf("listening on %s: %w", cfg.Addr, err)
	}

	server := grpcpkg.NewServer(
		grpcpkg.ChainUnaryInterceptor(
			loggerInterceptor(log),
			recoverInterceptor(),
			traceInterceptor(),
		),
	)
	accountsv1.RegisterAccountServiceServer(server, newAccountService(accountsService, idempotencyService))
	fundsv1.RegisterFundsServiceServer(server, newFundsService(depositsService, idempotencyService))
	reflection.Register(server)

	return &Server{server: server, listener: listener}, nil
}

func (s *Server) Addr() net.Addr {
	return s.listener.Addr()
}

func (s *Server) Serve() error {
	if err := s.server.Serve(s.listener); err != nil && !errors.Is(err, grpcpkg.ErrServerStopped) {
		return fmt.Errorf("serving grpc: %w", err)
	}
	return nil
}

func (s *Server) Shutdown(ctx context.Context) error {
	stopped := make(chan struct{})
	go func() {
		s.server.GracefulStop()
		close(stopped)
	}()

	select {
	case <-stopped:
		return nil
	case <-ctx.Done():
		s.server.Stop()
		return fmt.Errorf("graceful shutdown timed out: %w", ctx.Err())
	}
}

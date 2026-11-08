package grpc

import (
	"context"
	"errors"
	"fmt"
	"net"

	grpcpkg "google.golang.org/grpc"

	"github.com/dyingvoid/ledger/internal/core/accounts"
	"github.com/dyingvoid/ledger/internal/core/idempotency"
	"github.com/dyingvoid/ledger/internal/domain"
	accountsv1 "github.com/dyingvoid/ledger/pkg/proto/ledger/accounts/v1"
)

type Server struct {
	server   *grpcpkg.Server
	listener net.Listener
}

func NewServer(
	cfg Config,
	accountsService *accounts.Service,
	createAccount *idempotency.Runner[domain.Account],
) (*Server, error) {
	listener, err := net.Listen("tcp", cfg.Addr)
	if err != nil {
		return nil, fmt.Errorf("listening on %s: %w", cfg.Addr, err)
	}

	server := grpcpkg.NewServer()
	accountsv1.RegisterAccountServiceServer(server, newAccountService(accountsService, createAccount))

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

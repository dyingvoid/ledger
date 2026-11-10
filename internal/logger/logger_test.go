package logger

import (
	"context"
	"log/slog"
	"testing"
)

func TestNew(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name    string
		config  Config
		wantErr bool
	}{
		{name: "json", config: Config{Level: "debug", Format: "json"}},
		{name: "text", config: Config{Level: "warn", Format: "text"}},
		{name: "invalid format", config: Config{Level: "info", Format: "yaml"}, wantErr: true},
		{name: "invalid level", config: Config{Level: "loud", Format: "json"}, wantErr: true},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			t.Parallel()

			if _, err := New(test.config); (err != nil) != test.wantErr {
				t.Fatalf("New() error = %v, wantErr %t", err, test.wantErr)
			}
		})
	}
}

func TestContext(t *testing.T) {
	t.Parallel()

	ctx := context.Background()
	if got := FromContext(ctx); got != slog.Default() {
		t.Fatalf("FromContext() = %p, want default %p", got, slog.Default())
	}

	log := slog.New(slog.DiscardHandler)
	ctx = WithContext(ctx, log)

	if got := FromContext(ctx); got != log {
		t.Fatalf("FromContext() = %p, want %p", got, log)
	}
}

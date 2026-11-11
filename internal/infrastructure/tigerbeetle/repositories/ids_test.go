package repositories

import (
	"testing"

	"github.com/dyingvoid/ledger/internal/domain"
)

func TestToTigerbeetleID(t *testing.T) {
	t.Parallel()

	tests := []string{
		"1",
		"42",
		"255",
		"256",
		"340282366920938463463374607431768211454",
	}

	for _, value := range tests {
		t.Run(value, func(t *testing.T) {
			t.Parallel()

			id, err := domain.ParseID(value)
			if err != nil {
				t.Fatalf("ParseID(%q) error = %v", value, err)
			}

			if got := toTigerbeetleID(id).BigInt().String(); got != value {
				t.Fatalf("toTigerbeetleID(%s) = %s, want %s", value, got, value)
			}
		})
	}
}

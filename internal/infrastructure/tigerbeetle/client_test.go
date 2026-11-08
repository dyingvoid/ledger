package tigerbeetle

import (
	"testing"

	tigerbeetle_go "github.com/tigerbeetle/tigerbeetle-go"
)

func TestParseClusterID(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name  string
		value string
		want  tigerbeetle_go.Uint128
	}{
		{name: "zero", value: "0", want: tigerbeetle_go.ToUint128(0)},
		{name: "decimal", value: "42", want: tigerbeetle_go.ToUint128(42)},
		{name: "hex", value: "0xff", want: tigerbeetle_go.ToUint128(255)},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			t.Parallel()

			got, err := parseClusterID(test.value)
			if err != nil {
				t.Fatalf("parse cluster id: %v", err)
			}
			if got != test.want {
				t.Fatalf("cluster id = %s, want %s", got, test.want)
			}
		})
	}
}

func TestParseClusterIDErrors(t *testing.T) {
	t.Parallel()

	for _, value := range []string{"", "invalid", "0x100000000000000000000000000000000"} {
		t.Run(value, func(t *testing.T) {
			t.Parallel()

			if _, err := parseClusterID(value); err == nil {
				t.Fatalf("parse cluster id %q: expected error", value)
			}
		})
	}
}

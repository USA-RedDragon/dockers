package untouched

import (
	"errors"
	"fmt"
	"testing"
)

var errBase = errors.New("base")

func TestIs(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name string
		err  error
		want bool
	}{
		{"nil", nil, false},
		{"plain", errBase, false},
		{"wrapped", Wrap(errBase), true},
		{"wrapped again", fmt.Errorf("outer: %w", Wrap(errBase)), true},
		{"joined", errors.Join(errBase, Wrap(errBase)), true},
	}
	for _, tt := range tests {
		if got := Is(tt.err); got != tt.want {
			t.Errorf("%s: Is = %t, want %t", tt.name, got, tt.want)
		}
	}
	if Wrap(nil) != nil {
		t.Error("Wrap(nil) is not nil")
	}
	if err := Wrap(errBase); !errors.Is(err, errBase) || err.Error() != errBase.Error() {
		t.Errorf("Wrap changed the error: %v", err)
	}
}

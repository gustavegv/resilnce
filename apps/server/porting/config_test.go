package porting

import (
	"context"
	"errors"
	"os"
	"testing"
)

func TestApplyEnabledPolicy(t *testing.T) {
	for _, tc := range []struct {
		name, value string
		unset, want bool
	}{
		{name: "enabled by default", unset: true, want: true},
		{name: "empty configuration", want: true},
		{name: "explicitly enabled", value: "true", want: true},
		{name: "normalized true", value: " TRUE ", want: true},
		{name: "explicitly disabled", value: "false"},
		{name: "invalid configuration disables imports", value: "typo"},
	} {
		t.Run(tc.name, func(t *testing.T) {
			t.Setenv("PORTING_APPLY_ENABLED", tc.value)
			if tc.unset {
				if err := os.Unsetenv("PORTING_APPLY_ENABLED"); err != nil {
					t.Fatal(err)
				}
			}
			if got := ApplyEnabled(); got != tc.want {
				t.Fatalf("ApplyEnabled() = %v, want %v", got, tc.want)
			}
		})
	}
}

func TestDisabledApplyDoesNotAccessDatabase(t *testing.T) {
	s := &Service{}
	if _, err := s.StartRun(context.Background(), "operator", "apply", "", Options{}); !errors.Is(err, ErrApplyDisabled) {
		t.Fatalf("disabled run: %v", err)
	}
	if _, err := s.ImportOwner(context.Background(), nil, "a@example.com", Options{}); !errors.Is(err, ErrApplyDisabled) {
		t.Fatalf("disabled account import: %v", err)
	}
}

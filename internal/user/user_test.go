package user

import (
	"errors"
	"testing"
)

// TestCanAuthenticate is a table test: one slice of cases, one loop. Adding a new account status later means adding a line, not a function

func TestCanAuthenticate(t *testing.T) {
	cases := []struct {
		name string
		status Status
		wantErr error
	}{
		{ "active user can authenticate", StatusActive, nil },
		{"suspended user cannot", StatusSuspended, ErrAccountNotActive},
		{"deleted user cannot", StatusDeleted, ErrAccountNotActive},
		{"unknown status is rejected", Status("gibberish"), ErrAccountNotActive},
	}


	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			u := &User{Status: tc.status}

			err := u.CanAuthenticate()
			if !errors.Is(err, tc.wantErr) {
				t.Errorf("CanAuthenticate() with status %q = %v, want %v", tc.status, err, tc.wantErr)
			}
		})
	}
}
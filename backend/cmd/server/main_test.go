package main

import (
	"testing"

	"github.com/Sidd231/4yaar/backend/internal/config"
)

func TestValidateBootstrapCredentials(t *testing.T) {
	tests := []struct {
		name     string
		email    string
		password string
		wantErr  bool
	}{
		{
			name:     "valid credentials",
			email:    "admin@example.com",
			password: "a-secure-random-password-1234",
			wantErr:  false,
		},
		{
			name:     "missing email",
			email:    "",
			password: "a-secure-random-password-1234",
			wantErr:  true,
		},
		{
			name:     "missing password",
			email:    "admin@example.com",
			password: "",
			wantErr:  true,
		},
		{
			name:     "short password",
			email:    "admin@example.com",
			password: "short",
			wantErr:  true,
		},
		{
			name:     "legacy default password rejected",
			email:    "admin@example.com",
			password: config.LegacyAdminPassword,
			wantErr:  true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := ValidateBootstrapCredentials(tt.email, tt.password)
			if (err != nil) != tt.wantErr {
				t.Errorf("ValidateBootstrapCredentials() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

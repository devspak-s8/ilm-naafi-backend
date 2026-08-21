package security_test

import (
	"testing"
	"time"

	"github.com/ilmnafi/backend/internal/security"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestPasswordHash(t *testing.T) {
	ps := security.NewPasswordService()
	password := "TestPass123!"
	hash, err := ps.Hash(password)
	require.NoError(t, err)
	assert.NotEmpty(t, hash)
	assert.NotContains(t, hash, password)
	assert.Contains(t, hash, "argon2id")
}

func TestPasswordVerify(t *testing.T) {
	ps := security.NewPasswordService()
	password := "TestPass123!"
	hash, _ := ps.Hash(password)

	valid, err := ps.Verify(password, hash)
	require.NoError(t, err)
	assert.True(t, valid)

	valid, err = ps.Verify("wrongpassword", hash)
	require.NoError(t, err)
	assert.False(t, valid)
}

func TestPasswordValidation(t *testing.T) {
	tests := []struct {
		name    string
		pass    string
		wantErr bool
	}{
		{"valid password", "TestPass123!", false},
		{"too short", "Test1!", true},
		{"no uppercase", "testpass123!", true},
		{"no lowercase", "TESTPASS123!", true},
		{"no number", "TestPass!", true},
		{"no special", "TestPass123", true},
		{"too long", string(make([]byte, 129)), true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := security.ValidatePassword(tt.pass)
			if tt.wantErr {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
			}
		})
	}
}

func TestEmailValidation(t *testing.T) {
	tests := []struct {
		name    string
		email   string
		wantErr bool
	}{
		{"valid email", "test@example.com", false},
		{"invalid email", "invalid", true},
		{"empty email", "", true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := security.ValidateEmail(tt.email)
			if tt.wantErr {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
			}
		})
	}
}

func TestNameValidation(t *testing.T) {
	tests := []struct {
		name    string
		nameStr string
		wantErr bool
	}{
		{"valid name", "John Doe", false},
		{"too short", "J", true},
		{"invalid chars", "John123", true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := security.ValidateName(tt.nameStr)
			if tt.wantErr {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
			}
		})
	}
}

func TestGenerateSecureToken(t *testing.T) {
	ts := security.NewTokenService()
	token, err := ts.GenerateSecureToken(32)
	require.NoError(t, err)
	assert.Len(t, token, 43)
}

func TestGenerateSecureHash(t *testing.T) {
	ts := security.NewTokenService()
	hash := ts.GenerateSecureHash("test")
	assert.Len(t, hash, 8)
	assert.Equal(t, hash, ts.GenerateSecureHash("test"))
}

func TestNow(t *testing.T) {
	ts := security.NewTokenService()
	now := ts.Now()
	assert.True(t, now.Before(time.Now().UTC().Add(time.Minute)))
	assert.True(t, now.After(time.Now().Add(-time.Minute).UTC()))
}

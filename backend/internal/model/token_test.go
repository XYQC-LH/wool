package model

import (
	"testing"

	"github.com/google/uuid"
	"github.com/stretchr/testify/require"
)

func TestTokenEffectiveTenantID_PreferExplicitTenant(t *testing.T) {
	token := &Token{
		UserID:   uuid.New(),
		TenantID: "tenant-alpha",
	}

	require.Equal(t, "tenant-alpha", token.EffectiveTenantID())
}

func TestTokenEffectiveTenantID_FallbackToUserID(t *testing.T) {
	userID := uuid.New()
	token := &Token{
		UserID: userID,
	}

	require.Equal(t, userID.String(), token.EffectiveTenantID())
}

func TestTokenEffectiveTenantID_HandleNil(t *testing.T) {
	var token *Token
	require.Equal(t, "", token.EffectiveTenantID())
}

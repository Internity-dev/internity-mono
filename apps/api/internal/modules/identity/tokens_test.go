package identity

import (
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestNewOpaqueToken_UniqueAndHighEntropy(t *testing.T) {
	seen := make(map[string]bool)
	for i := 0; i < 1000; i++ {
		tok, err := newOpaqueToken()
		require.NoError(t, err)
		require.NotEmpty(t, tok)
		assert.False(t, seen[tok], "generated a duplicate token")
		seen[tok] = true
	}
}

func TestHashToken_DeterministicAndDistinct(t *testing.T) {
	a := hashToken("raw-token-a")
	aAgain := hashToken("raw-token-a")
	b := hashToken("raw-token-b")

	assert.Equal(t, a, aAgain, "hashing the same input twice must be deterministic")
	assert.NotEqual(t, a, b, "different inputs must hash differently")
	assert.Len(t, a, 64, "expected a hex-encoded sha256 (32 bytes = 64 hex chars)")
}

func TestNewInviteCodeString_AlphabetAndLength(t *testing.T) {
	for i := 0; i < 200; i++ {
		code, err := newInviteCodeString()
		require.NoError(t, err)
		assert.Len(t, code, 8)
		assert.Equal(t, strings.ToUpper(code), code, "invite codes should be uppercase")
		for _, ch := range code {
			assert.Contains(t, inviteCodeAlphabet, string(ch), "code contains a character outside the unambiguous alphabet")
		}
	}
}

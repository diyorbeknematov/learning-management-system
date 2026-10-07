package password_test

import (
	"testing"

	"github.com/diyorbeknematov/lms/pkg/password"
	"github.com/stretchr/testify/require"
)

func TestHashAndCompare(t *testing.T) {
	hash, err := password.Hash("secret123")

	require.NoError(t, err)
	require.NotEqual(t, "secret123", hash)
	require.True(t, password.Compare(hash, "secret123"))
	require.False(t, password.Compare(hash, "wrong"))
}

func TestHash_DiffersEveryTime(t *testing.T) {
	first, err := password.Hash("secret123")
	require.NoError(t, err)

	second, err := password.Hash("secret123")
	require.NoError(t, err)

	require.NotEqual(t, first, second)
}

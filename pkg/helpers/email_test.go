package helpers_test

import (
	"strings"
	"testing"

	"github.com/diyorbeknematov/lms/pkg/helpers"
	"github.com/stretchr/testify/require"
)

func TestNormalizeEmail(t *testing.T) {
	cases := map[string]string{
		"ali@example.com":            "ali@example.com",
		"  Ali@Example.COM  ":        "ali@example.com",
		"first.last+tag@example.org": "first.last+tag@example.org",
		"a@sub.example.co.uk":        "a@sub.example.co.uk",
	}

	for input, want := range cases {
		got, err := helpers.NormalizeEmail(input)
		require.NoError(t, err, input)
		require.Equal(t, want, got, input)
	}
}

func TestNormalizeEmail_Rejects(t *testing.T) {
	for _, input := range []string{
		"",
		"   ",
		"plainaddress",
		"@example.com",
		"ali@",
		"ali@example",
		"ali@.com",
		"ali@example.",
		"ali example@example.com",
		"Ali <ali@example.com>",
		"ali@example.com, bob@example.com",
		"ali@@example.com",
		strings.Repeat("a", 250) + "@x.com",
	} {
		_, err := helpers.NormalizeEmail(input)
		require.ErrorIs(t, err, helpers.ErrInvalidEmail, input)
	}
}

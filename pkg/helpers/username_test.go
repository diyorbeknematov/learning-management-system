package helpers_test

import (
	"strings"
	"testing"

	"github.com/diyorbeknematov/lms/pkg/helpers"
	"github.com/stretchr/testify/require"
)

func TestNormalizeUsername(t *testing.T) {
	cases := map[string]string{
		"ali":                      "ali",
		"  ali  ":                  "ali",
		"Ali":                      "Ali",
		"ali_vali":                 "ali_vali",
		"ali.vali-2":               "ali.vali-2",
		"007":                      "007",
		"abc":                      "abc",
		strings.Repeat("a", 30):    strings.Repeat("a", 30),
		"A1":                       "",
		"":                         "",
		"ab":                       "",
		strings.Repeat("a", 31):    "",
		"_ali":                     "",
		".ali":                     "",
		"-ali":                     "",
		"ali vali":                 "",
		"ali@vali":                 "",
		"ali/vali":                 "",
		"<script>":                 "",
		"аdmin":                    "",
		"Алишер":                   "",
		"ali​":                     "",
		"alı":                      "",
		"ali\nvali":                "",
		"ali';DROP TABLE users;--": "",
		"o'zbek":                   "",
		"ali\x00":                  "",
	}

	for input, want := range cases {
		got, err := helpers.NormalizeUsername(input)

		if want == "" {
			require.ErrorIs(t, err, helpers.ErrInvalidUsername, "%q", input)
			continue
		}

		require.NoError(t, err, "%q", input)
		require.Equal(t, want, got, "%q", input)
	}
}

func TestNormalizeUsername_KeepsTheCase(t *testing.T) {
	upper, err := helpers.NormalizeUsername("Ali")
	require.NoError(t, err)

	lower, err := helpers.NormalizeUsername("ali")
	require.NoError(t, err)

	require.NotEqual(t, upper, lower)
}

package password_test

import (
	"errors"
	"strings"
	"testing"

	"github.com/diyorbeknematov/lms/pkg/password"
	"github.com/stretchr/testify/require"
)

func TestValidate_Accepts(t *testing.T) {
	for _, plain := range []string{
		"Passw0rd!",
		"Abcdef1#",
		"correct-Horse-battery-9",
		"Parol123$",
		"Пароль1!аБ",
		"Ab1_" + strings.Repeat("x", 68),
	} {
		require.NoError(t, password.Validate(plain), plain)
	}
}

func TestValidate_Rejects(t *testing.T) {
	cases := map[string]string{
		"":                               "at least 8 characters",
		"Ab1!":                           "at least 8 characters",
		"Abcdef1":                        "a special character",
		"password1!":                     "a capital letter",
		"PASSWORD1!":                     "a small letter",
		"Password!!":                     "a digit",
		"Password1":                      "a special character",
		"Passw0rd 1":                     "a special character",
		"Ab1_" + strings.Repeat("x", 69): "at most 72 bytes",
	}

	for plain, problem := range cases {
		err := password.Validate(plain)
		require.Error(t, err, plain)

		var policy *password.PolicyError
		require.True(t, errors.As(err, &policy), plain)
		require.Contains(t, policy.Problems, problem, plain)
	}
}

func TestValidate_NamesEveryMissingPart(t *testing.T) {
	var policy *password.PolicyError

	require.True(t, errors.As(password.Validate("abc"), &policy))
	require.ElementsMatch(t, []string{"at least 8 characters", "a capital letter", "a digit", "a special character"}, policy.Problems)
	require.Contains(t, policy.Error(), "a capital letter")
}

func TestValidate_TheLimitIsBytesNotCharacters(t *testing.T) {
	// 40 two-byte letters are 80 bytes: bcrypt would refuse them
	plain := "Ab1!" + strings.Repeat("ё", 40)

	var policy *password.PolicyError

	require.True(t, errors.As(password.Validate(plain), &policy))
	require.Contains(t, policy.Problems, "at most 72 bytes")

	_, err := password.Hash("Ab1!" + strings.Repeat("x", 68))
	require.NoError(t, err)
}

package core

import (
	"errors"

	"github.com/diyorbeknematov/lms/pkg/apperror"
	"github.com/diyorbeknematov/lms/pkg/helpers"
	"github.com/diyorbeknematov/lms/pkg/password"
)

// CheckPassword checks a new password against the password rules. The error
// says what is missing. The handler checks the same rules; this is the second
// lock for the service.
func CheckPassword(plain, op string) error {
	err := password.Validate(plain)
	if err == nil {
		return nil
	}

	var policy *password.PolicyError

	if errors.As(err, &policy) {
		return apperror.InvalidInput("service", op, policy.Error(), apperror.ErrInvalidInput)
	}

	return apperror.InvalidInput("service", op, "invalid password", apperror.ErrInvalidInput)
}

// NormalizeEmail returns the email in lower case, or an invalid input error.
func NormalizeEmail(email, op string) (string, error) {
	normalized, err := helpers.NormalizeEmail(email)
	if err != nil {
		return "", apperror.InvalidInput("service", op, "invalid email address", apperror.ErrInvalidInput)
	}

	return normalized, nil
}

// NormalizeUsername returns the trimmed username, or an invalid input error. The
// case is kept.
func NormalizeUsername(username, op string) (string, error) {
	normalized, err := helpers.NormalizeUsername(username)
	if err != nil {
		return "", apperror.InvalidInput(
			"service",
			op,
			"a username has 3 to 30 Latin letters, digits, dots, dashes or underscores",
			apperror.ErrInvalidInput,
		)
	}

	return normalized, nil
}

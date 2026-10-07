package helpers

import (
	"errors"
	"net/mail"
	"strings"
)

const maxEmailLength = 254

var ErrInvalidEmail = errors.New("invalid email address")

// NormalizeEmail trims an email address and makes it lower case, so Ali@x.com
// and ali@x.com are one address. It returns ErrInvalidEmail for anything that
// is not a plain address: a display name ("Ali <a@x.com>"), a missing domain
// part, spaces, or more than 254 characters.
func NormalizeEmail(email string) (string, error) {
	email = strings.ToLower(strings.TrimSpace(email))

	if email == "" || len(email) > maxEmailLength {
		return "", ErrInvalidEmail
	}

	address, err := mail.ParseAddress(email)
	if err != nil || address.Address != email {
		return "", ErrInvalidEmail
	}

	local, domain, found := strings.Cut(email, "@")
	if !found || local == "" || !strings.Contains(domain, ".") || strings.HasPrefix(domain, ".") || strings.HasSuffix(domain, ".") {
		return "", ErrInvalidEmail
	}

	return email, nil
}

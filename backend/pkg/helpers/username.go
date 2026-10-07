package helpers

import (
	"errors"
	"regexp"
	"strings"
)

var ErrInvalidUsername = errors.New("invalid username")

// usernamePattern allows Latin letters, digits and _ . - only, 3 to 30
// characters, starting with a letter or a digit. Letters of other alphabets
// are left out on purpose: a Cyrillic "а" looks like a Latin "a", and somebody
// could register "аdmin" to pass for "admin".
var usernamePattern = regexp.MustCompile(`^[A-Za-z0-9][A-Za-z0-9._-]{2,29}$`)

// NormalizeUsername trims a username and checks it. The case is kept: Ali and
// ali are two different usernames.
func NormalizeUsername(username string) (string, error) {
	username = strings.TrimSpace(username)

	if !usernamePattern.MatchString(username) {
		return "", ErrInvalidUsername
	}

	return username, nil
}

package password

import (
	"strings"
	"unicode"
	"unicode/utf8"
)

const (
	MinLength = 8
	// MaxBytes is the longest password bcrypt can hash; it gives an error for
	// anything longer.
	MaxBytes = 72
)

// PolicyError lists what a password is missing.
type PolicyError struct {
	Problems []string
}

func (e *PolicyError) Error() string {
	return "the password must have " + strings.Join(e.Problems, ", ")
}

// Validate checks a new password: at least 8 characters, and a capital letter,
// a small letter, a digit and a special character (a punctuation mark or a
// symbol; a space does not count). It returns a *PolicyError that names every
// missing part, or nil.
func Validate(plain string) error {
	var problems []string

	if utf8.RuneCountInString(plain) < MinLength {
		problems = append(problems, "at least 8 characters")
	}

	if len(plain) > MaxBytes {
		problems = append(problems, "at most 72 bytes")
	}

	var upper, lower, digit, special bool

	for _, r := range plain {
		switch {
		case unicode.IsUpper(r):
			upper = true
		case unicode.IsLower(r):
			lower = true
		case unicode.IsDigit(r):
			digit = true
		case unicode.IsPunct(r) || unicode.IsSymbol(r):
			special = true
		}
	}

	if !upper {
		problems = append(problems, "a capital letter")
	}

	if !lower {
		problems = append(problems, "a small letter")
	}

	if !digit {
		problems = append(problems, "a digit")
	}

	if !special {
		problems = append(problems, "a special character")
	}

	if len(problems) == 0 {
		return nil
	}

	return &PolicyError{Problems: problems}
}

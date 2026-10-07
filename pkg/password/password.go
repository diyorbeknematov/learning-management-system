package password

import "golang.org/x/crypto/bcrypt"

// Hash returns the bcrypt hash of a plain password.
func Hash(plain string) (string, error) {
	hash, err := bcrypt.GenerateFromPassword([]byte(plain), bcrypt.DefaultCost)
	if err != nil {
		return "", err
	}

	return string(hash), nil
}

// Compare reports whether the plain password matches the bcrypt hash.
func Compare(hash, plain string) bool {
	return bcrypt.CompareHashAndPassword([]byte(hash), []byte(plain)) == nil
}

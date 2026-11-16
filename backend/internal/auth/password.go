package auth

import (
	"errors"
	"sync"

	"golang.org/x/crypto/bcrypt"
)

// bcryptCost is 10, the library default, on purpose. The free Render instance
// has a fraction of a CPU; at cost 12 a single login takes seconds.
const bcryptCost = 10

// HashPassword returns a bcrypt hash suitable for app_user.password_hash.
// Length rules are checked earlier, in domain.ValidateRegistration; bcrypt
// itself refuses anything over 72 bytes, so that limit is enforced twice.
func HashPassword(password string) (string, error) {
	hash, err := bcrypt.GenerateFromPassword([]byte(password), bcryptCost)
	if err != nil {
		return "", err
	}
	return string(hash), nil
}

// CheckPassword reports whether password matches hash. Any error — a wrong
// password or a malformed hash — is a plain "no".
func CheckPassword(hash, password string) bool {
	return bcrypt.CompareHashAndPassword([]byte(hash), []byte(password)) == nil
}

var (
	dummyHashOnce sync.Once
	dummyHash     string
)

// BurnPasswordCheck does the same amount of work as a real CheckPassword and
// throws the result away. Login calls it when the email is unknown, so that
// "no such account" and "wrong password" take the same time and an attacker
// cannot use response times to find out which emails are registered.
func BurnPasswordCheck(password string) {
	dummyHashOnce.Do(func() {
		h, err := bcrypt.GenerateFromPassword([]byte("not a real password"), bcryptCost)
		if err != nil {
			panic(errors.New("auth: cannot create dummy bcrypt hash: " + err.Error()))
		}
		dummyHash = string(h)
	})
	CheckPassword(dummyHash, password)
}

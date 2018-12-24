// Package passhash is a utility package to standardize and abstract away how passwords are hashed and compared.
package passhash

import "golang.org/x/crypto/bcrypt"

var (
	// MinCost is the minimum allowable cost as passed in to GenerateFromPasswordWithCost
	MinCost = bcrypt.MinCost
	// MaxCost is the maximum allowable cost as passed in to GenerateFromPasswordWithCost
	MaxCost = bcrypt.MaxCost
	// DefaultCost is the cost used for GenerateFromPassword
	DefaultCost = bcrypt.DefaultCost
)

// GenerateFromPassword returns the hash of the password.
func GenerateFromPassword(password []byte) ([]byte, error) {
	return GenerateFromPasswordWithCost(password, DefaultCost)
}

// GenerateFromPasswordWithCost returns the hash of the password with a custom cost
func GenerateFromPasswordWithCost(password []byte, cost int) ([]byte, error) {
	return bcrypt.GenerateFromPassword(password, cost)
}

// CompareHashAndPassword compares a hashed password with its possible
// plaintext equivalent. Returns nil on success, or an error on failure.
func CompareHashAndPassword(hash, password []byte) error {
	return bcrypt.CompareHashAndPassword(hash, password)
}

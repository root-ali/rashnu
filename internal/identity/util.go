package identity

import (
	"crypto/rand"
	"errors"
	"math/big"

	"golang.org/x/crypto/bcrypt"
)

func hashPassword(password string) (string, error) {
	hashedPassword, err := bcrypt.GenerateFromPassword([]byte(password), bcrypt.DefaultCost)
	if err != nil {
		return "", errors.New("ERROR_GENERATING_HASH_PASSWORD")
	}
	return string(hashedPassword), nil
}

func checkPassword(hashedPassword, plainPassword string) error {
	return bcrypt.CompareHashAndPassword([]byte(hashedPassword), []byte(plainPassword))
}

func generateRandomPassword(length int) (string, error) {
	if length <= 0 {
		return "", errors.New("password length must be greater than zero")
	}
	const charSet = "abcdefghijklmnopqrstuvwxyzABCDEFGHIJKLMNOPQRSTUVWXYZ0123456789!@#$%^&*()"
	password := make([]byte, length)
	for i := 0; i < length; i++ {
		randomIndex, err := rand.Int(rand.Reader, big.NewInt(int64(len(charSet))))
		if err != nil {
			return "", err
		}
		password[i] = charSet[randomIndex.Int64()]
	}

	return string(password), nil
}

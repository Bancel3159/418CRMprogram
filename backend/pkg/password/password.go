package password

import (
	"errors"

	"golang.org/x/crypto/bcrypt"
)

// Hash 哈希密码
func Hash(plain string) (string, error) {
	if len(plain) < 6 {
		return "", errors.New("密码长度至少 6 位")
	}
	hashed, err := bcrypt.GenerateFromPassword([]byte(plain), bcrypt.DefaultCost)
	if err != nil {
		return "", err
	}
	return string(hashed), nil
}

// Verify 验证密码
func Verify(hashed, plain string) bool {
	return bcrypt.CompareHashAndPassword([]byte(hashed), []byte(plain)) == nil
}
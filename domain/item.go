package domain

import (
	"crypto/sha256"
	"encoding/hex"
)

type User struct {
	ID       string `json:"id"`
	Email    string `json:"email"`
	Password string `json:"password"`
	Name     string `json:"name"`
	TwoFactorSecret string `json:"two_factor_secret"`
	TwoFactorVerified bool `json:"two_factor_verified"`
	TwoFactorEnabled bool `json:"two_factor_enabled"`
	TwoFactorRecoveryCodes string `json:"two_factor_recovery_codes"`
}

func (u *User) ValidatePassword(password string) bool {
	hash := sha256.Sum256([]byte(password))
	return hex.EncodeToString(hash[:]) == u.Password
}
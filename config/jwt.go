package config

import (
	"log"
	"os"
)

// JWTSecret returns the signing key for JWTs, read once from the
// JWT_SECRET environment variable. The app refuses to start if it's
// missing or still set to the old hardcoded placeholder value, so this
// mistake can't silently ship again.
func JWTSecret() []byte {
	secret := os.Getenv("JWT_SECRET")

	if secret == "" {
		log.Fatal("JWT_SECRET environment variable is not set. Set it before starting the app.")
	}

	if secret == "your-super-secret-key-change-this-later" {
		log.Fatal("JWT_SECRET is still set to the old placeholder value. Generate a real secret.")
	}

	return []byte(secret)
}
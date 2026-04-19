package config

import (
	"os"
	"strconv"
)

type Config struct {
	Port int

	EmailHost string
	EmailPort int
	EmailUser string
	EmailPass string

	SMSApiKey string
	SMSApiURL string
}

func Load() *Config {
	return &Config {
		Port: getEnvInt("PORT", 8080),
		EmailHost: getEnv("EMAIL_HOST", ""),
		EmailPort: getEnvInt("EMAIL_PORT", 587),
		EmailUser: getEnv("EMAIL_USER", ""),
		EmailPass: getEnv("EMAIL_PASS", ""),
		SMSApiKey: getEnv("SMS_API_KEY", ""),
		SMSApiURL: getEnv("SMS_API_URL", ""),
	}
}

func getEnv(key, fallback string) string {
	if v:= os.Getenv(key); v != "" {
		return v
	}
	return fallback
}

func getEnvInt(key string, fallback int) int {
	v := os.Getenv(key)
	if v == "" {
		return fallback
	}
	n, err := strconv.Atoi(v)
	if err != nil {
		return fallback
	}
	return n
}


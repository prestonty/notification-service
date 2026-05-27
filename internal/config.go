package config

import (
	"os"
	"strconv"
)

type Config struct {
	Port int

	ResendAPIKey string
	ResendFrom   string

	SMSApiKey string
	SMSApiURL string
}

func Load() *Config {
	return &Config{
		Port:          getEnvInt("PORT", 8080),
		ResendAPIKey: getEnv("RESEND_API_KEY", ""),
		ResendFrom:   getEnv("RESEND_FROM", ""),
		SMSApiKey:     getEnv("SMS_API_KEY", ""),
		SMSApiURL:     getEnv("SMS_API_URL", ""),
	}
}

func getEnv(key, fallback string) string {
	if v := os.Getenv(key); v != "" {
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

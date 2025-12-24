package config

import (
	"log"
	"os"
)

type Config struct {
	KisAppKey     string
	KisAppSecret  string
	KisAccountNum string
	KisBaseURL    string // Real: https://openapi.koreainvestment.com:9443, Virtual: https://openapivts.koreainvestment.com:29443
}

func Load() *Config {
	return &Config{
		KisAppKey:     getEnv("KIS_APP_KEY", ""),
		KisAppSecret:  getEnv("KIS_APP_SECRET", ""),
		KisAccountNum: getEnv("KIS_ACCOUNT_NUM", ""),
		KisBaseURL:    getEnv("KIS_BASE_URL", "https://openapi.koreainvestment.com:9443"),
	}
}

func getEnv(key, fallback string) string {
	if value, exists := os.LookupEnv(key); exists {
		return value
	}
	if fallback == "" {
		log.Printf("Warning: Environment variable %s not set", key)
	}
	return fallback
}

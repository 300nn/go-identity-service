package config

import "os"

type Config struct {
	HttpPort    string
	AppName     string
	Version     string
	Environment string
	LogLevel    string
}

func Load() Config {
	return Config{
		HttpPort:    getEnv("HTTP_PORT", "8080"),
		AppName:     getEnv("APP_NAME", "go-crud-api"),
		Version:     getEnv("VERSION", "dev"),
		Environment: getEnv("APP_ENV", "local"),
		LogLevel:    getEnv("LOG_LEVEL", "info"),
	}
}

func getEnv(key, defaultValue string) string {
	val := os.Getenv(key)
	if val == "" {
		return defaultValue
	}
	return val
}

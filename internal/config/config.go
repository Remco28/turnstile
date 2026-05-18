package config

import "os"

type Config struct {
	DBPath     string
	ListenAddr string
}

func Load() Config {
	cfg := Config{
		DBPath:     envOrDefault("TURNSTILE_DB_PATH", "./turnstile.db"),
		ListenAddr: envOrDefault("TURNSTILE_LISTEN_ADDR", "127.0.0.1:7432"),
	}
	return cfg
}

func envOrDefault(key, fallback string) string {
	if value := os.Getenv(key); value != "" {
		return value
	}
	return fallback
}

package config

import (
	"log"
	"os"

	"github.com/joho/godotenv"
)

type Config struct {
	DBDSN         string
	ServerPort    string
	SessionSecret string
}

func Load() *Config {
	_ = godotenv.Load()

	cfg := &Config{
		DBDSN:         os.Getenv("DB_DSN"),
		ServerPort:    os.Getenv("SERVER_PORT"),
		SessionSecret: os.Getenv("SESSION_SECRET"),
	}

	if cfg.DBDSN == "" {
		log.Fatal("DB_DSN is not set")
	}
	if cfg.ServerPort == "" {
		cfg.ServerPort = "8080"
	}
	if cfg.SessionSecret == "" {
		log.Fatal("SESSION_SECRET is not set")
	}

	return cfg
}

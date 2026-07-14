package config

import (
	"fmt"
	"os"

	"github.com/joho/godotenv"
)

// Config holds every value the application needs to run
// It is parseed once at startup and injected as a dependency

type Config struct{
	AppEnv		string
	Port		string
	DatabaseURL	string
}


// Load reads configuration from the environment and validate it
// It returns an error if any required value is missing so the process can refuse to start rather than fail on runtime.

func Load() (*Config, error) {
	// .eenv is a local-development convenience. In staging/Production the environment is injected by the deploy target, so a missing .env file is not an error

	_ = godotenv.Load()
	cfg := &Config{
		AppEnv: 		getEnv("APP_ENV", "development"),
		Port: 			getEnv("PORT", "8080"),
		DatabaseURL: 	os.Getenv("DATABASE_URL"),
	}

	if err := cfg.validate(); err != nil {
		return nil, err
	}
	return cfg, nil
}

// validate enforces that every required field is present.
func (c *Config) validate() error {
	if c.DatabaseURL == "" {
		return fmt.Errorf("config: DATABASE_URL is required")
	}
	return nil
}

// getEnv returns the value of keey, or fallback if it isi unset.
func getEnv(key, fallback string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return fallback
}
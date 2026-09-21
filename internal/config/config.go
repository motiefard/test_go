package config

import (
	"bufio"
	"fmt"
	"os"
	"strings"
	"time"
)

type Config struct {
	HTTPAddr          string
	SQLitePath        string
	MigrationsPath    string
	CORSOrigin        string
	ZarinHubBaseURL   string
	ZarinHubTimeout   time.Duration
	ZarinHubToken     string
	ZarinHubGUID      string
	ZarinHubPassword  string
}

func Load(envFile string) (Config, error) {
	if envFile != "" {
		if err := loadDotEnv(envFile); err != nil && !os.IsNotExist(err) {
			return Config{}, err
		}
	}

	timeout, err := time.ParseDuration(getenv("ZARINHUB_TIMEOUT", "8s"))
	if err != nil {
		return Config{}, fmt.Errorf("invalid ZARINHUB_TIMEOUT: %w", err)
	}

	cfg := Config{
		HTTPAddr:         getenv("HTTP_ADDR", ":8080"),
		SQLitePath:       getenv("SQLITE_PATH", "./data/app.db"),
		MigrationsPath:   getenv("MIGRATIONS_PATH", "./migrations"),
		CORSOrigin:       getenv("CORS_ORIGIN", "http://localhost:3000"),
		ZarinHubBaseURL:  strings.TrimRight(getenv("ZARINHUB_BASE_URL", "https://zarin-hub.com"), "/"),
		ZarinHubTimeout:  timeout,
		ZarinHubToken:    os.Getenv("ZARINHUB_TOKEN"),
		ZarinHubGUID:     os.Getenv("ZARINHUB_GUID"),
		ZarinHubPassword: os.Getenv("ZARINHUB_API_PASSWORD"),
	}
	return cfg, nil
}

func (c Config) BearerToken() string {
	if strings.TrimSpace(c.ZarinHubToken) != "" {
		return strings.TrimSpace(c.ZarinHubToken)
	}
	return strings.TrimSpace(c.ZarinHubGUID)
}

func getenv(key, fallback string) string {
	v := strings.TrimSpace(os.Getenv(key))
	if v == "" {
		return fallback
	}
	return v
}

func loadDotEnv(path string) error {
	f, err := os.Open(path)
	if err != nil {
		return err
	}
	defer f.Close()

	scanner := bufio.NewScanner(f)
	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())
		if line == "" || strings.HasPrefix(line, "#") {
			continue
		}
		key, value, ok := strings.Cut(line, "=")
		if !ok {
			continue
		}
		key = strings.TrimSpace(key)
		value = strings.TrimSpace(value)
		if len(value) >= 2 {
			if (value[0] == '"' && value[len(value)-1] == '"') || (value[0] == '\'' && value[len(value)-1] == '\'') {
				value = value[1 : len(value)-1]
			}
		}
		if os.Getenv(key) == "" {
			_ = os.Setenv(key, value)
		}
	}
	return scanner.Err()
}

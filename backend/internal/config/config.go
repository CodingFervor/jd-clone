package config

import "os"

// Config holds runtime configuration sourced from environment variables with
// sensible defaults so the server runs with zero configuration.
type Config struct {
	Port           int
	DBPath         string
	AllowedOrigins string
	JWTSecret      string
}

func Load() Config {
	return Config{
		Port:           envInt("PORT", 8080),
		DBPath:         envStr("DB_PATH", "data/app.db"),
		AllowedOrigins: envStr("ALLOWED_ORIGINS", "*"),
		JWTSecret:      envStr("JWT_SECRET", "jd-clone-dev-secret"),
	}
}

func envStr(key, def string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return def
}

func envInt(key string, def int) int {
	if v := os.Getenv(key); v != "" {
		n := 0
		for _, c := range v {
			if c < '0' || c > '9' {
				return def
			}
			n = n*10 + int(c-'0')
		}
		return n
	}
	return def
}

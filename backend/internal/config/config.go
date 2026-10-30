// Package config loads server configuration from the process environment,
// with an optional .env file fallback (real environment variables win).
// Secrets only ever come from the environment — never from code (CLAUDE.md §5).
package config

import (
	"fmt"
	"os"
	"strings"
)

type Config struct {
	DatabaseURL  string
	JWTSecret    []byte
	CookieSecure bool   // true in production (HTTPS); false only for local dev
	Port         string // backend listen port
	StaticDir    string // built SPA directory served by the binary; "" disables static serving
}

// Load reads configuration, falling back to a .env file next to the backend
// module or at the repository root.
func Load() (*Config, error) {
	LoadDotEnv(".env")
	LoadDotEnv("../.env")

	cfg := &Config{
		DatabaseURL:  strings.TrimSpace(os.Getenv("DATABASE_URL")),
		CookieSecure: strings.EqualFold(strings.TrimSpace(os.Getenv("COOKIE_SECURE")), "true"),
		Port:         strings.TrimSpace(os.Getenv("PORT")),
	}

	secret := strings.TrimSpace(os.Getenv("JWT_SECRET"))
	if len(secret) < 32 {
		return nil, fmt.Errorf("JWT_SECRET must be set to at least 32 bytes (see .env.example)")
	}
	cfg.JWTSecret = []byte(secret)

	if cfg.DatabaseURL == "" {
		return nil, fmt.Errorf("DATABASE_URL must be set (see .env.example)")
	}
	if cfg.Port == "" {
		cfg.Port = "8080"
	}

	staticDir, err := resolveStaticDir(strings.TrimSpace(os.Getenv("STATIC_DIR")))
	if err != nil {
		return nil, err
	}
	cfg.StaticDir = staticDir
	return cfg, nil
}

// resolveStaticDir picks the built-frontend directory: an explicit STATIC_DIR
// wins (and must exist — a wrong path is a startup error, not a silently
// broken server), otherwise the conventional Vite output locations are
// probed. An empty result disables SPA serving.
func resolveStaticDir(explicit string) (string, error) {
	if explicit != "" {
		st, err := os.Stat(explicit)
		if err != nil {
			return "", fmt.Errorf("STATIC_DIR %q does not exist (set it to the built-frontend directory, or unset it to disable SPA serving)", explicit)
		}
		if !st.IsDir() {
			return "", fmt.Errorf("STATIC_DIR %q is not a directory", explicit)
		}
		return explicit, nil
	}
	for _, candidate := range []string{"../frontend/dist", "frontend/dist"} {
		if st, err := os.Stat(candidate); err == nil && st.IsDir() {
			return candidate, nil
		}
	}
	return "", nil
}

// LoadDotEnv applies KEY=VALUE lines from path for variables not already
// set. Exported so the migrate command shares the same configuration flow.
func LoadDotEnv(path string) {
	data, err := os.ReadFile(path)
	if err != nil {
		return // no .env — process environment only
	}
	for _, line := range strings.Split(string(data), "\n") {
		line = strings.TrimSpace(strings.TrimSuffix(line, "\r"))
		if line == "" || strings.HasPrefix(line, "#") {
			continue
		}
		key, value, ok := strings.Cut(line, "=")
		if !ok {
			continue
		}
		key = strings.TrimSpace(key)
		value = strings.TrimSpace(value)
		if len(value) >= 2 && (value[0] == '"' && value[len(value)-1] == '"' || value[0] == '\'' && value[len(value)-1] == '\'') {
			value = value[1 : len(value)-1]
		}
		if os.Getenv(key) == "" {
			_ = os.Setenv(key, value)
		}
	}
}

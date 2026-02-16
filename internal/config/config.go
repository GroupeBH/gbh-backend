package config

import (
	"net/url"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"time"
)

type Config struct {
	Env                   string
	MongoURI              string
	MongoDB               string
	ServerAddr            string
	FrontendOrigins       []string
	RateLimitAppointments int
	RateLimitContact      int
	RateLimitWindowSec    int
	RedisURL              string
	RedisAddr             string
	RedisPassword         string
	RedisDB               int
	CacheTTLSeconds       int
	AdminAPIKey           string
	AdminSetupKey         string
	AdminUser             string
	AdminPassword         string
	JWTSecret             string
	AccessTTLMinutes      int
	RefreshTTLMinutes     int
	CookieSecure          bool
	Timezone              *time.Location
	BrevoAPIKey           string
	BrevoSenderEmail      string
	BrevoSenderName       string
	BrevoSandbox          bool
}

func getEnv(key, fallback string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return fallback
}

func getEnvInt(key string, fallback int) int {
	if v := os.Getenv(key); v != "" {
		if i, err := strconv.Atoi(v); err == nil {
			return i
		}
	}
	return fallback
}

func Load() (*Config, error) {
	loadDotEnv(".env")
	loc, err := time.LoadLocation(getEnv("TZ", "Africa/Kinshasa"))
	if err != nil {
		return nil, err
	}

	mongoURI := getEnv("MONGO_URI", "mongodb://localhost:27017/gbh")
	mongoDB := getEnv("MONGO_DB", "")
	if mongoDB == "" {
		mongoDB = mongoDBFromURI(mongoURI)
	}
	if mongoDB == "" {
		mongoDB = "gbh"
	}

	frontendOrigins := parseOrigins(getEnv("FRONTEND_ORIGINS", ""))
	if len(frontendOrigins) == 0 {
		frontendOrigins = parseOrigins(getEnv("FRONTEND_ORIGIN", "httpS://www.gbh.sarl"))
	}

	cfg := &Config{
		Env:                   getEnv("APP_ENV", "development"),
		MongoURI:              mongoURI,
		MongoDB:               mongoDB,
		ServerAddr:            getEnv("SERVER_ADDR", ":8080"),
		FrontendOrigins:       frontendOrigins,
		RateLimitAppointments: getEnvInt("RATE_LIMIT_APPOINTMENTS", 10),
		RateLimitContact:      getEnvInt("RATE_LIMIT_CONTACT", 5),
		RateLimitWindowSec:    getEnvInt("RATE_LIMIT_WINDOW_SEC", 60),
		RedisURL:              getEnv("REDIS_URL", ""),
		RedisAddr:             getEnv("REDIS_ADDR", ""),
		RedisPassword:         getEnv("REDIS_PASSWORD", ""),
		RedisDB:               getEnvInt("REDIS_DB", 0),
		CacheTTLSeconds:       getEnvInt("CACHE_TTL_SECONDS", 60),
		AdminAPIKey:           getEnv("ADMIN_API_KEY", ""),
		AdminSetupKey:         getEnv("ADMIN_SETUP_KEY", ""),
		AdminUser:             getEnv("ADMIN_USER", "admin"),
		AdminPassword:         getEnv("ADMIN_PASSWORD", ""),
		JWTSecret:             getEnv("JWT_SECRET", ""),
		AccessTTLMinutes:      getEnvInt("ACCESS_TTL_MINUTES", 15),
		RefreshTTLMinutes:     getEnvInt("REFRESH_TTL_MINUTES", 43200),
		CookieSecure:          getEnv("COOKIE_SECURE", "false") == "true",
		Timezone:              loc,
		BrevoAPIKey:           getEnv("BREVO_API_KEY", ""),
		BrevoSenderEmail:      getEnv("BREVO_SENDER_EMAIL", ""),
		BrevoSenderName:       getEnv("BREVO_SENDER_NAME", ""),
		BrevoSandbox:          getEnv("BREVO_SANDBOX", "false") == "true",
	}

	return cfg, nil
}

func parseOrigins(value string) []string {
	if strings.TrimSpace(value) == "" {
		return nil
	}
	parts := strings.Split(value, ",")
	out := make([]string, 0, len(parts))
	for _, p := range parts {
		p = strings.TrimSpace(p)
		p = strings.TrimRight(p, "/")
		if p != "" {
			out = append(out, p)
		}
	}
	return out
}

func mongoDBFromURI(uri string) string {
	u, err := url.Parse(uri)
	if err != nil {
		return ""
	}
	db := strings.Trim(u.Path, "/")
	if db == "" {
		return ""
	}
	// mongodb URIs sometimes include extra path segments; we only support the first one as db name.
	if idx := strings.Index(db, "/"); idx >= 0 {
		db = db[:idx]
	}
	return db
}

func loadDotEnv(path string) {
	data, err := os.ReadFile(path)
	if err != nil && !filepath.IsAbs(path) {
		if alt := findEnvFile(path); alt != "" {
			data, err = os.ReadFile(alt)
		}
	}
	if err != nil {
		return
	}
	lines := strings.Split(string(data), "\n")
	for _, line := range lines {
		line = strings.TrimSpace(line)
		if line == "" || strings.HasPrefix(line, "#") {
			continue
		}
		parts := strings.SplitN(line, "=", 2)
		if len(parts) != 2 {
			continue
		}
		key := strings.TrimSpace(parts[0])
		val := strings.TrimSpace(parts[1])
		if key == "" {
			continue
		}
		if _, exists := os.LookupEnv(key); exists {
			continue
		}
		_ = os.Setenv(key, val)
	}
}

func findEnvFile(name string) string {
	wd, err := os.Getwd()
	if err != nil {
		return ""
	}
	dir := wd
	for {
		envPath := filepath.Join(dir, name)
		if fileExists(envPath) {
			return envPath
		}
		if fileExists(filepath.Join(dir, "go.mod")) {
			return ""
		}
		parent := filepath.Dir(dir)
		if parent == dir {
			return ""
		}
		dir = parent
	}
}

func fileExists(path string) bool {
	info, err := os.Stat(path)
	if err != nil {
		return false
	}
	return !info.IsDir()
}

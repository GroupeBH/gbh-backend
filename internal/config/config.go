package config

import (
	"os"
	"strconv"
	"strings"
	"time"
)

type Config struct {
	Env                 string
	MongoURI            string
	MongoDB             string
	ServerAddr          string
	FrontendOrigin      string
	RateLimitAppointments int
	RateLimitContact      int
	RateLimitWindowSec    int
	RedisAddr           string
	RedisPassword       string
	RedisDB             int
	CacheTTLSeconds     int
	Timezone            *time.Location
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

	cfg := &Config{
		Env:                   getEnv("APP_ENV", "development"),
		MongoURI:              getEnv("MONGO_URI", "mongodb://localhost:27017"),
		MongoDB:               getEnv("MONGO_DB", "gbh"),
		ServerAddr:            getEnv("SERVER_ADDR", ":8080"),
		FrontendOrigin:        getEnv("FRONTEND_ORIGIN", "http://localhost:3000"),
		RateLimitAppointments: getEnvInt("RATE_LIMIT_APPOINTMENTS", 10),
		RateLimitContact:      getEnvInt("RATE_LIMIT_CONTACT", 5),
		RateLimitWindowSec:    getEnvInt("RATE_LIMIT_WINDOW_SEC", 60),
		RedisAddr:             getEnv("REDIS_ADDR", ""),
		RedisPassword:         getEnv("REDIS_PASSWORD", ""),
		RedisDB:               getEnvInt("REDIS_DB", 0),
		CacheTTLSeconds:       getEnvInt("CACHE_TTL_SECONDS", 60),
		Timezone:              loc,
	}

	return cfg, nil
}

func loadDotEnv(path string) {
	data, err := os.ReadFile(path)
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

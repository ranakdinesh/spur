package config

import (
    "os"
    "strconv"
    "strings"

    "github.com/joho/godotenv"
)

type HTTPConfig struct {
    Enable bool
    Port   int
    EnableCORS bool
}

type GRPCConfig struct {
    Enable bool
    Port   int
}

type LogConfig struct {
    Level string
    LoggerServiceURL string
    Env   string // development|production
}

type PostgresConfig struct {
    DSN string
}

type RedisConfig struct {
    Addr     string
    Password string
    DB       int
}

type Config struct {
    AppName string
    HTTP    HTTPConfig
    GRPC    GRPCConfig
    Log     LogConfig
    Postgres PostgresConfig
    Redis    RedisConfig
}

func getenv(key, def string) string {
    if v := os.Getenv(key); v != "" {
        return v
    }
    return def
}

func getbool(key string, def bool) bool {
    v := strings.ToLower(getenv(key, ""))
    if v == "true" || v == "1" || v == "yes" || v == "y" {
        return true
    }
    if v == "false" || v == "0" || v == "no" || v == "n" {
        return false
    }
    return def
}

func getint(key string, def int) int {
    s := getenv(key, "")
    if s == "" { return def }
    if i, err := strconv.Atoi(s); err == nil {
        return i
    }
    return def
}

func Load() (*Config, error) {
    _ = godotenv.Load() // optional

    cfg := &Config{
        AppName: getenv("APP_NAME", "spur"),
        HTTP: HTTPConfig{
            Enable:    getbool("HTTP_ENABLE", true),
            Port:      getint("HTTP_PORT", 8080),
            EnableCORS: getbool("HTTP_ENABLE_CORS", true),
        },
        GRPC: GRPCConfig{
            Enable: getbool("GRPC_ENABLE", true),
            Port:   getint("GRPC_PORT", 9090),
        },
        Log: LogConfig{
            Level: getenv("LOG_LEVEL", "info"),
            LoggerServiceURL: getenv("LOGGER_SERVICE_URL", ""),
            Env: getenv("APP_ENV", "development"),
        },
        Postgres: PostgresConfig{
            DSN: getenv("PG_DSN", "postgres://postgres:postgres@localhost:5432/postgres?sslmode=disable"),
        },
        Redis: RedisConfig{
            Addr: getenv("REDIS_ADDR", "localhost:6379"),
            Password: getenv("REDIS_PASSWORD", ""),
            DB: getint("REDIS_DB", 0),
        },
    }
    return cfg, nil
}

package renderer

import (
	"os"
	"strconv"
	"time"
)

type CacheMode string

const (
	CacheNone     CacheMode = "none"
	CachePartial  CacheMode = "partial"
	CacheFullAnon CacheMode = "full_anon"
	CacheFull     CacheMode = "full"
)

type SecurityConfig struct {
	CSRFSecret []byte
}

type CacheConfig struct {
	Mode       CacheMode
	MaxEntries int
	TTL        time.Duration
}

type Config struct {
	TemplateRoot string
	DefaultTheme string
	Cache        CacheConfig
	Security     SecurityConfig
	Logger       interface{ Printf(string, ...any) }
}

func (c Config) withDefaults() Config {
	if c.TemplateRoot == "" {
		c.TemplateRoot = "./templates"
	}
	if c.DefaultTheme == "" {
		c.DefaultTheme = "default"
	}
	if c.Cache.MaxEntries <= 0 {
		c.Cache.MaxEntries = 256
	}
	if c.Cache.TTL <= 0 {
		c.Cache.TTL = 300 * time.Second
	}
	return c
}

func LoadConfigFromEnv() Config {
	cfg := Config{}
	cfg.TemplateRoot = getenvDefault("RENDERER_TEMPLATE_ROOT", "./templates")
	cfg.DefaultTheme = getenvDefault("RENDERER_DEFAULT_THEME", "default")

	switch getenvDefault("RENDERER_CACHE_MODE", "partial") {
	case "none":
		cfg.Cache.Mode = CacheNone
	case "full":
		cfg.Cache.Mode = CacheFull
	case "full_anon":
		cfg.Cache.Mode = CacheFullAnon
	default:
		cfg.Cache.Mode = CachePartial
	}
	if v := os.Getenv("RENDERER_CACHE_MAX_ENTRIES"); v != "" {
		if i, err := strconv.Atoi(v); err == nil && i > 0 {
			cfg.Cache.MaxEntries = i
		}
	}
	if v := os.Getenv("RENDERER_CACHE_TTL_SECONDS"); v != "" {
		if i, err := strconv.Atoi(v); err == nil && i > 0 {
			cfg.Cache.TTL = time.Duration(i) * time.Second
		}
	}
	if sec := os.Getenv("RENDERER_CSRF_SECRET"); sec != "" {
		cfg.Security.CSRFSecret = []byte(sec)
	}
	return cfg.withDefaults()
}

func getenvDefault(k, def string) string {
	if v := os.Getenv(k); v != "" {
		return v
	}
	return def
}

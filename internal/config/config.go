package config

import (
	"fmt"
	"path/filepath"

	"github.com/urfave/cli/v2"
)

// Flags defines all CLI flags for the application
var Flags = []cli.Flag{
	&cli.StringFlag{
		Name:    "address",
		Aliases: []string{"a"},
		EnvVars: []string{"ADDRESS"},
		Value:   "0.0.0.0",
		Usage:   "Address to bind to",
	},
	&cli.IntFlag{
		Name:    "port",
		Aliases: []string{"p"},
		EnvVars: []string{"PORT"},
		Value:   8080,
		Usage:   "Port to listen on",
	},
	&cli.StringFlag{
		Name:    "directory",
		Aliases: []string{"d"},
		EnvVars: []string{"DIRECTORY"},
		Value:   ".",
		Usage:   "Directory to serve",
	},
	&cli.BoolFlag{
		Name:    "spa",
		EnvVars: []string{"SPA_MODE"},
		Value:   true,
		Usage:   "Enable SPA mode (serve index.html for unknown routes)",
	},
	&cli.StringFlag{
		Name:    "fallback",
		EnvVars: []string{"FALLBACK_FILE"},
		Value:   "index.html",
		Usage:   "Fallback file for SPA mode",
	},
	&cli.BoolFlag{
		Name:    "gzip",
		EnvVars: []string{"GZIP"},
		Value:   false,
		Usage:   "Enable gzip compression",
	},
	&cli.BoolFlag{
		Name:    "brotli",
		EnvVars: []string{"BROTLI"},
		Value:   false,
		Usage:   "Enable brotli compression (preferred over gzip)",
	},
	&cli.Int64Flag{
		Name:    "threshold",
		EnvVars: []string{"THRESHOLD"},
		Value:   1024,
		Usage:   "Minimum file size (bytes) for compression",
	},
	&cli.Int64Flag{
		Name:    "cache-max-age",
		EnvVars: []string{"CACHE_MAX_AGE"},
		Value:   604800,
		Usage:   "Cache-Control max-age in seconds (set -1 to disable)",
	},
	&cli.StringSliceFlag{
		Name:    "no-cache-paths",
		EnvVars: []string{"NO_CACHE_PATHS"},
		Usage:   "Paths to set Cache-Control: no-store",
	},
	&cli.StringSliceFlag{
		Name:    "no-compress",
		EnvVars: []string{"NO_COMPRESS"},
		Usage:   "File extensions to skip compression (e.g., .jpg,.png)",
	},
	&cli.BoolFlag{
		Name:    "cache",
		EnvVars: []string{"CACHE"},
		Value:   true,
		Usage:   "Enable in-memory file caching",
	},
	&cli.IntFlag{
		Name:    "cache-size",
		EnvVars: []string{"CACHE_SIZE"},
		Value:   50 * 1024,
		Usage:   "LRU cache size in bytes",
	},
	&cli.BoolFlag{
		Name:    "logger",
		EnvVars: []string{"LOGGER"},
		Value:   false,
		Usage:   "Enable request logging",
	},
	&cli.BoolFlag{
		Name:    "log-pretty",
		EnvVars: []string{"LOG_PRETTY"},
		Value:   false,
		Usage:   "Pretty print logs (instead of JSON)",
	},
	&cli.BoolFlag{
		Name:    "health",
		EnvVars: []string{"HEALTH_ENDPOINT"},
		Value:   true,
		Usage:   "Enable /healthz endpoint",
	},
	&cli.BoolFlag{
		Name:    "security-headers",
		EnvVars: []string{"SECURITY_HEADERS"},
		Value:   true,
		Usage:   "Add security headers (X-Frame-Options, etc.)",
	},
	&cli.BoolFlag{
		Name:    "cors",
		EnvVars: []string{"CORS"},
		Value:   false,
		Usage:   "Enable CORS headers",
	},
	&cli.StringFlag{
		Name:    "cors-origin",
		EnvVars: []string{"CORS_ORIGIN"},
		Value:   "*",
		Usage:   "CORS Access-Control-Allow-Origin value",
	},
}

// Config holds all configuration values
type Config struct {
	Address         string
	Port            int
	Directory       string
	SpaMode         bool
	FallbackFile    string
	Gzip            bool
	Brotli          bool
	Threshold       int64
	CacheMaxAge     int64
	NoCachePaths    []string
	NoCompress      []string
	CacheEnabled    bool
	CacheSize       int
	Logger          bool
	LogPretty       bool
	HealthEndpoint  bool
	SecurityHeaders bool
	CORS            bool
	CORSOrigin      string
}

// FromContext creates Config from CLI context
func FromContext(c *cli.Context) (*Config, error) {
	directory, err := filepath.Abs(c.String("directory"))
	if err != nil {
		return nil, fmt.Errorf("invalid directory path: %w", err)
	}

	return &Config{
		Address:         c.String("address"),
		Port:            c.Int("port"),
		Directory:       directory,
		SpaMode:         c.Bool("spa"),
		FallbackFile:    c.String("fallback"),
		Gzip:            c.Bool("gzip"),
		Brotli:          c.Bool("brotli"),
		Threshold:       c.Int64("threshold"),
		CacheMaxAge:     c.Int64("cache-max-age"),
		NoCachePaths:    c.StringSlice("no-cache-paths"),
		NoCompress:      c.StringSlice("no-compress"),
		CacheEnabled:    c.Bool("cache"),
		CacheSize:       c.Int("cache-size"),
		Logger:          c.Bool("logger"),
		LogPretty:       c.Bool("log-pretty"),
		HealthEndpoint:  c.Bool("health"),
		SecurityHeaders: c.Bool("security-headers"),
		CORS:            c.Bool("cors"),
		CORSOrigin:      c.String("cors-origin"),
	}, nil
}

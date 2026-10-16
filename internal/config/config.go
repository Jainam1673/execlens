package config

import (
	"flag"
	"os"
	"strconv"
)

// Config represents runtime configuration settings for ExecLens.
type Config struct {
	Host          string
	Port          int
	DBPath        string
	Concurrency   int
	BenchmarksDir string
	FrontendDir   string
}

// Load loads configuration from environment variables and command line flags.
func Load() Config {
	cfg := Config{
		Host:          getEnv("EXECLENS_HOST", "0.0.0.0"),
		Port:          getEnvInt("EXECLENS_PORT", 8080),
		DBPath:        getEnv("EXECLENS_DB_PATH", "execlens.db"),
		Concurrency:   getEnvInt("EXECLENS_CONCURRENCY", 1),
		BenchmarksDir: getEnv("EXECLENS_BENCHMARKS_DIR", "benchmarks"),
		FrontendDir:   getEnv("EXECLENS_FRONTEND_DIR", "frontend"),
	}
	return cfg
}

func getEnv(key, def string) string {
	if val := os.Getenv(key); val != "" {
		return val
	}
	return def
}

func getEnvInt(key string, def int) int {
	if val := os.Getenv(key); val != "" {
		if i, err := strconv.Atoi(val); err == nil {
			return i
		}
	}
	return def
}

// BindFlags registers CLI flags to override configuration.
func (c *Config) BindFlags(fs *flag.FlagSet) {
	fs.StringVar(&c.Host, "host", c.Host, "HTTP server host")
	fs.IntVar(&c.Port, "port", c.Port, "HTTP server port")
	fs.StringVar(&c.DBPath, "db", c.DBPath, "SQLite database file path")
	fs.IntVar(&c.Concurrency, "concurrency", c.Concurrency, "Max concurrent benchmark jobs")
	fs.StringVar(&c.BenchmarksDir, "benchmarks", c.BenchmarksDir, "Benchmarks directory path")
	fs.StringVar(&c.FrontendDir, "frontend", c.FrontendDir, "Frontend directory path")
}

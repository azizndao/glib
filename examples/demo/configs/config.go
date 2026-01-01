package configs

import (
	"fmt"
	"time"
)

// @Config
type Config struct {
	Server struct {
		Host    string        `env:"APP_HOST" default:"0.0.0.0"`
		Port    int           `env:"APP_PORT" default:"8080"`
		Env     string        `env:"APP_ENV" default:"development"`
		Timeout time.Duration `env:"ROUTER_TIMEOUT" default:"60"` // seconds, 0 = no timeout
	}

	Database struct {
		Filename string `env:"DB_FILENAME" default:"data.db"`
	}
}

func (c *Config) Addr() string {
	return fmt.Sprintf("%s:%d", c.Server.Host, c.Server.Port)
}

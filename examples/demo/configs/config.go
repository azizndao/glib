package configs

// Config holds application configuration
// @Config
type Config struct {
	Server struct {
		Port    int    `env:"APP_PORT" default:"8080"`
		Env     string `env:"APP_ENV" default:"development"`
		Timeout int    `env:"ROUTER_TIMEOUT" default:"60"` // seconds, 0 = no timeout
	}

	CORS struct {
		Enabled          bool     `env:"CORS_ENABLED"`
		AllowedOrigins   []string `env:"CORS_ALLOWED_ORIGINS"`
		AllowedMethods   []string `env:"CORS_ALLOWED_METHODS"`
		AllowedHeaders   []string `env:"CORS_ALLOWED_HEADERS"`
		ExposedHeaders   []string `env:"CORS_EXPOSED_HEADERS"`
		AllowCredentials bool     `env:"CORS_ALLOW_CREDENTIALS"`
		MaxAge           int      `env:"CORS_MAX_AGE" default:"300"` // seconds
	}

	Database struct {
		Filename string `env:"DB_FILENAME" default:"db.sqlite"`
	}
}

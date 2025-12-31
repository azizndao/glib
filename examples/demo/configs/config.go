package configs

// Config holds application configuration
// @Config
type Config struct {
	App struct {
		Port int    `env:"APP_PORT" default:"8080"`
		Env  string `env:"APP_ENV" default:"development"`
	}
	Database struct {
		Host     string `env:"DB_HOST" default:"localhost"`
		Port     int    `env:"DB_PORT" default:"5432"`
		Name     string `env:"DB_NAME" default:"app"`
		User     string `env:"DB_USER" default:"postgres"`
		Password string `env:"DB_PASSWORD" required:"true"`
	}
}

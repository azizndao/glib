package configs

// @Config
type RedisConfig struct {
	Host     string `env:"REDIS_HOST" default:"localhost"`
	Port     int    `env:"REDIS_PORT" default:"6379"`
	Password string `env:"REDIS_PASSWORD"`
}

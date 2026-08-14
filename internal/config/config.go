package config

import "time"

type Config struct {
	Server   ServerConfig
	Postgres PostgresConfig
}

type ServerConfig struct {
	Host         string        `env:"HOST" env-default:"localhost"`
	Port         int           `env:"PORT" env-default:"8080"`
	IdleTimeout  time.Duration `env:"IDLE_TIMEOUT" env-default:"10s"`
	ReadTimeout  time.Duration `env:"READ_TIMEOUT" env-default:"10s"`
	WriteTimeout time.Duration `env:"WRITE_TIMEOUT" env-default:"10s"`
}

type PostgresConfig struct {
	URL string `env:"POSTGRES_URL" env-required:"true"`
}

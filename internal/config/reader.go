package config

import (
	"github.com/ilyakaznacheev/cleanenv"
	"github.com/joho/godotenv"
)

func InitEnv(filenames ...string) error {
	err := godotenv.Load(filenames...)
	if err != nil {
		return err
	}
	return nil
}

func ReadEnv() (*Config, error) {
	config := new(Config)
	err := cleanenv.ReadEnv(config)
	if err != nil {
		return nil, err
	}
	return config, nil
}

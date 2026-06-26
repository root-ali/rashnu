package config

import (
	"fmt"
	"strings"

	"github.com/knadh/koanf/parsers/yaml"
	"github.com/knadh/koanf/providers/env"
	"github.com/knadh/koanf/providers/file"
	"github.com/knadh/koanf/providers/structs"
	"github.com/knadh/koanf/v2"
)

func NewConfig(path string) (*Config, error) {
	var cfg Config

	k := koanf.New(".")

	if err := k.Load(structs.Provider(Config{
		Http: HttpConfig{
			Host: "0.0.0.0",
			Port: "8080",
		},
		Logger: LoggerConfig{
			Level:    "info",
			Env:      "development",
			Encoding: "json",
		},
		JWT: JWTConfig{
			Secret: "secret",
		},
		Database: DatabaseConfig{
			Host:               "localhost",
			Port:               "5432",
			DB:                 "rashnu",
			User:               "postgres",
			Pass:               "postgres",
			SSLMode:            false,
			MaxConnections:     20,
			MinConnections:     2,
			MaxIdleConnections: 10,
			MaxOpenConnections: 10,
			ConnMaxLifetime:    30,
			ConnMaxIdleTime:    5,
		},
	}, "koanf"), nil); err != nil {
		return nil, fmt.Errorf("load config: %w", err)
	}

	if err := k.Load(file.Provider(path), yaml.Parser()); err != nil {
		return nil, fmt.Errorf("load config file: %w", err)
	}

	if err := k.Load(env.Provider("RASHNU_", ".", func(s string) string {
		return strings.Replace(s, "RASHNU_", "", 1)
	}), nil); err != nil {
		return nil, err
	}

	if err := k.Unmarshal("", &cfg); err != nil {
		return nil, err
	}

	fmt.Println("config File loaded", cfg)

	return &cfg, nil
}

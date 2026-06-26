package config

type Config struct {
	Database DatabaseConfig `env:"DATABASE" koanf:"database"`
	Logger   LoggerConfig   `env:"LOGGER" koanf:"logger"`
	Http     HttpConfig     `env:"HTTP" koanf:"http"`
	JWT      JWTConfig      `env:"JWT" koanf:"jwt"`
}

type DatabaseConfig struct {
	Host               string `env:"DATABASE_HOST" koanf:"host"`
	Port               string `env:"DATABASE_PORT" koanf:"port"`
	DB                 string `env:"DATABASE_NAME" koanf:"name"`
	User               string `env:"DATABASE_USER" koanf:"user"`
	Pass               string `env:"DATABASE_PASS" koanf:"pass"`
	SSLMode            bool   `env:"DATABASE_SSL_MODE" koanf:"ssl_mode"`
	MaxConnections     int    `env:"DATABASE_MAX_CONNECTIONS" koanf:"max_connections"`
	MinConnections     int    `env:"DATABASE_MIN_CONNECTIONS" koanf:"min_connections"`
	MaxIdleConnections int    `env:"DATABASE_MAX_IDLE_CONNECTIONS" koanf:"max_idle_connections"`
	MaxOpenConnections int    `env:"DATABASE_MAX_OPEN_CONNECTIONS" koanf:"max_open_connections"`
	ConnMaxLifetime    int    `env:"DATABASE_CONN_MAX_LIFETIME" koanf:"conn_max_lifetime"`
	ConnMaxIdleTime    int    `env:"DATABASE_CONN_MAX_IDLE_TIME" koanf:"conn_max_idle_time"`
}

type HttpConfig struct {
	Host           string `env:"HTTP_HOST" koanf:"host"`
	Port           string `env:"HTTP_PORT" koanf:"port"`
	ReadTimeout    int    `env:"HTTP_READ_TIMEOUT" koanf:"read_timeout"`
	WriteTimeout   int    `env:"HTTP_WRITE_TIMEOUT" koanf:"write_timeout"`
	IdleTimeout    int    `env:"HTTP_IDLE_TIMEOUT" koanf:"idle_timeout"`
	MaxHeaderBytes int    `env:"HTTP_MAX_HEADER_BYTES" koanf:"max_header_bytes"`
}

type JWTConfig struct {
	Secret string `env:"JWT_SECRET" koanf:"secret"`
}

type LoggerConfig struct {
	Encoding string `env:"LOG_ENCODING" koanf:"encoding"`
	Level    string `env:"LOG_LEVEL" koanf:"level"`
	Env      string `env:"ENV" koanf:"env"`
}

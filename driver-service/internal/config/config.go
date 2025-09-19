package config

import (
	"fmt"
	"strings"
	"time"

	"github.com/spf13/viper"
)

// Config структура конфигурации приложения
type Config struct {
	Server   ServerConfig   `mapstructure:"server"`
	Database DatabaseConfig `mapstructure:"database"`
	Redis    RedisConfig    `mapstructure:"redis"`
	NATS     NATSConfig     `mapstructure:"nats"`
	Logger   LoggerConfig   `mapstructure:"logger"`
	External ExternalConfig `mapstructure:"external"`
	Metrics  MetricsConfig  `mapstructure:"metrics"`
}

// ServerConfig конфигурация HTTP и gRPC серверов
type ServerConfig struct {
	HTTPPort    int           `mapstructure:"http_port"`
	GRPCPort    int           `mapstructure:"grpc_port"`
	MetricsPort int           `mapstructure:"metrics_port"`
	Timeout     time.Duration `mapstructure:"timeout"`
	Environment string        `mapstructure:"environment"`
}

// DatabaseConfig конфигурация PostgreSQL
type DatabaseConfig struct {
	Host            string        `mapstructure:"host"`
	Port            int           `mapstructure:"port"`
	User            string        `mapstructure:"user"`
	Password        string        `mapstructure:"password"`
	Database        string        `mapstructure:"database"`
	SSLMode         string        `mapstructure:"ssl_mode"`
	MaxOpenConns    int           `mapstructure:"max_open_conns"`
	MaxIdleConns    int           `mapstructure:"max_idle_conns"`
	ConnMaxLifetime time.Duration `mapstructure:"conn_max_lifetime"`
}

// RedisConfig конфигурация Redis
type RedisConfig struct {
	Host        string        `mapstructure:"host"`
	Port        int           `mapstructure:"port"`
	Password    string        `mapstructure:"password"`
	Database    int           `mapstructure:"database"`
	MaxRetries  int           `mapstructure:"max_retries"`
	PoolSize    int           `mapstructure:"pool_size"`
	IdleTimeout time.Duration `mapstructure:"idle_timeout"`
}

// NATSConfig конфигурация NATS
type NATSConfig struct {
	URL             string        `mapstructure:"url"`
	ClusterID       string        `mapstructure:"cluster_id"`
	ClientID        string        `mapstructure:"client_id"`
	ConnectTimeout  time.Duration `mapstructure:"connect_timeout"`
	ReconnectDelay  time.Duration `mapstructure:"reconnect_delay"`
	MaxReconnect    int           `mapstructure:"max_reconnect"`
	PingInterval    time.Duration `mapstructure:"ping_interval"`
	MaxPingsOut     int           `mapstructure:"max_pings_out"`
}

// LoggerConfig конфигурация логгера
type LoggerConfig struct {
	Level      string `mapstructure:"level"`
	Format     string `mapstructure:"format"`
	OutputPath string `mapstructure:"output_path"`
}

// ExternalConfig конфигурация внешних сервисов
type ExternalConfig struct {
	GIBDDAPI GIBDDAPIConfig `mapstructure:"gibdd_api"`
	MapsAPI  MapsAPIConfig  `mapstructure:"maps_api"`
	SMSAPI   SMSAPIConfig   `mapstructure:"sms_api"`
	S3       S3Config       `mapstructure:"s3"`
}

// GIBDDAPIConfig конфигурация API ГИБДД
type GIBDDAPIConfig struct {
	BaseURL string        `mapstructure:"base_url"`
	APIKey  string        `mapstructure:"api_key"`
	Timeout time.Duration `mapstructure:"timeout"`
}

// MapsAPIConfig конфигурация карт
type MapsAPIConfig struct {
	BaseURL string        `mapstructure:"base_url"`
	APIKey  string        `mapstructure:"api_key"`
	Timeout time.Duration `mapstructure:"timeout"`
}

// SMSAPIConfig конфигурация SMS
type SMSAPIConfig struct {
	BaseURL  string        `mapstructure:"base_url"`
	APIKey   string        `mapstructure:"api_key"`
	From     string        `mapstructure:"from"`
	Timeout  time.Duration `mapstructure:"timeout"`
}

// S3Config конфигурация S3
type S3Config struct {
	Endpoint        string `mapstructure:"endpoint"`
	AccessKeyID     string `mapstructure:"access_key_id"`
	SecretAccessKey string `mapstructure:"secret_access_key"`
	BucketName      string `mapstructure:"bucket_name"`
	Region          string `mapstructure:"region"`
	UseSSL          bool   `mapstructure:"use_ssl"`
}

// MetricsConfig конфигурация метрик
type MetricsConfig struct {
	Enabled bool   `mapstructure:"enabled"`
	Path    string `mapstructure:"path"`
}

// LoadConfig загружает конфигурацию из переменных окружения и файлов
func LoadConfig() (*Config, error) {
	viper.SetConfigName("config")
	viper.SetConfigType("yaml")
	viper.AddConfigPath(".")
	viper.AddConfigPath("./configs")

	// Установка значений по умолчанию
	setDefaults()

	// Настройка чтения переменных окружения
	viper.AutomaticEnv()
	viper.SetEnvKeyReplacer(strings.NewReplacer(".", "_"))
	viper.SetEnvPrefix("DRIVER_SERVICE")

	// Чтение конфигурационного файла
	if err := viper.ReadInConfig(); err != nil {
		if _, ok := err.(viper.ConfigFileNotFoundError); !ok {
			return nil, fmt.Errorf("error reading config file: %w", err)
		}
		// Файл конфигурации не найден, используем переменные окружения и значения по умолчанию
	}

	var config Config
	if err := viper.Unmarshal(&config); err != nil {
		return nil, fmt.Errorf("error unmarshaling config: %w", err)
	}

	return &config, nil
}

// setDefaults устанавливает значения по умолчанию
func setDefaults() {
	// Server
	viper.SetDefault("server.http_port", 8001)
	viper.SetDefault("server.grpc_port", 9001)
	viper.SetDefault("server.metrics_port", 9002)
	viper.SetDefault("server.timeout", "30s")
	viper.SetDefault("server.environment", "development")

	// Database
	viper.SetDefault("database.host", "localhost")
	viper.SetDefault("database.port", 5432)
	viper.SetDefault("database.user", "postgres")
	viper.SetDefault("database.password", "postgres")
	viper.SetDefault("database.database", "driver_service")
	viper.SetDefault("database.ssl_mode", "disable")
	viper.SetDefault("database.max_open_conns", 25)
	viper.SetDefault("database.max_idle_conns", 25)
	viper.SetDefault("database.conn_max_lifetime", "5m")

	// Redis
	viper.SetDefault("redis.host", "localhost")
	viper.SetDefault("redis.port", 6379)
	viper.SetDefault("redis.password", "")
	viper.SetDefault("redis.database", 0)
	viper.SetDefault("redis.max_retries", 3)
	viper.SetDefault("redis.pool_size", 10)
	viper.SetDefault("redis.idle_timeout", "5m")

	// NATS
	viper.SetDefault("nats.url", "nats://localhost:4222")
	viper.SetDefault("nats.cluster_id", "driver-service-cluster")
	viper.SetDefault("nats.client_id", "driver-service")
	viper.SetDefault("nats.connect_timeout", "30s")
	viper.SetDefault("nats.reconnect_delay", "2s")
	viper.SetDefault("nats.max_reconnect", -1)
	viper.SetDefault("nats.ping_interval", "20s")
	viper.SetDefault("nats.max_pings_out", 2)

	// Logger
	viper.SetDefault("logger.level", "info")
	viper.SetDefault("logger.format", "json")
	viper.SetDefault("logger.output_path", "stdout")

	// External APIs
	viper.SetDefault("external.gibdd_api.timeout", "30s")
	viper.SetDefault("external.maps_api.timeout", "10s")
	viper.SetDefault("external.sms_api.timeout", "15s")

	// S3
	viper.SetDefault("external.s3.region", "us-east-1")
	viper.SetDefault("external.s3.use_ssl", true)

	// Metrics
	viper.SetDefault("metrics.enabled", true)
	viper.SetDefault("metrics.path", "/metrics")
}

// GetDSN возвращает строку подключения к базе данных
func (c *DatabaseConfig) GetDSN() string {
	return fmt.Sprintf("host=%s port=%d user=%s password=%s dbname=%s sslmode=%s",
		c.Host, c.Port, c.User, c.Password, c.Database, c.SSLMode)
}

// GetRedisAddr возвращает адрес Redis
func (c *RedisConfig) GetRedisAddr() string {
	return fmt.Sprintf("%s:%d", c.Host, c.Port)
}

// Validate проверяет корректность конфигурации
func (c *Config) Validate() error {
	if c.Server.HTTPPort <= 0 || c.Server.HTTPPort > 65535 {
		return fmt.Errorf("invalid HTTP port: %d", c.Server.HTTPPort)
	}

	if c.Server.GRPCPort <= 0 || c.Server.GRPCPort > 65535 {
		return fmt.Errorf("invalid gRPC port: %d", c.Server.GRPCPort)
	}

	if c.Database.Host == "" {
		return fmt.Errorf("database host is required")
	}

	if c.Database.User == "" {
		return fmt.Errorf("database user is required")
	}

	if c.Database.Database == "" {
		return fmt.Errorf("database name is required")
	}

	if c.NATS.URL == "" {
		return fmt.Errorf("NATS URL is required")
	}

	return nil
}
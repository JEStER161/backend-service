package configs

import (
	"errors"
	"fmt"
	"io"
	"net"
	"net/url"
	"os"
	"strconv"
	"strings"
	"time"

	"gopkg.in/yaml.v3"
)

type Duration struct {
	time.Duration
}

// UnmarshalYAML преобразует строковую длительность из YAML в time.Duration
func (d *Duration) UnmarshalYAML(node *yaml.Node) error {
	if node.Kind != yaml.ScalarNode {
		return errors.New("duration must be a string")
	}
	parsed, err := time.ParseDuration(node.Value)
	if err != nil {
		return fmt.Errorf("invalid duration %q: %w", node.Value, err)
	}
	d.Duration = parsed
	return nil
}

type Config struct {
	Mode           string         `yaml:"mode"`
	Logging        Logging        `yaml:"logging"`
	Listen         Listen         `yaml:"listen"`
	Infrastructure Infrastructure `yaml:"infrastructure"`
	Application    Application    `yaml:"application"`
}

type Logging struct {
	Level              string `yaml:"level"`
	Encoding           string `yaml:"encoding"`
	Development        bool   `yaml:"development"`
	DisableCaller      bool   `yaml:"disableCaller"`
	DisableStacktrace  bool   `yaml:"disableStacktrace"`
	RequestIDFieldName string `yaml:"requestIDFieldName"`
	ApplicationName    string `yaml:"applicationName"`
}

type Listen struct {
	HTTP HTTP `yaml:"http"`
}

type HTTP struct {
	IsEnabled                bool     `yaml:"isEnabled"`
	Host                     string   `yaml:"host"`
	Port                     int      `yaml:"port"`
	MaxRequestProcessingTime Duration `yaml:"maxRequestProcessingTime"`
	ReadHeaderTimeout        Duration `yaml:"readHeaderTimeout"`
	ReadTimeout              Duration `yaml:"readTimeout"`
	WriteTimeout             Duration `yaml:"writeTimeout"`
	IdleTimeout              Duration `yaml:"idleTimeout"`
	ShutdownTimeout          Duration `yaml:"shutdownTimeout"`
}

// Address собирает адрес HTTP-сервера из хоста и порта
func (c HTTP) Address() string {
	return net.JoinHostPort(c.Host, strconv.Itoa(c.Port))
}

type Infrastructure struct {
	Postgresql Postgresql `yaml:"postgresql"`
}

type Postgresql struct {
	RW PostgresqlInstance `yaml:"rw"`
}

type PostgresqlInstance struct {
	IsEnabled       bool                  `yaml:"isEnabled"`
	Host            string                `yaml:"host"`
	Port            int                   `yaml:"port"`
	DBName          string                `yaml:"dbName"`
	UserName        string                `yaml:"userName"`
	Password        string                `yaml:"password"`
	SSLMode         string                `yaml:"sslMode"`
	ApplicationName string                `yaml:"applicationName"`
	Connections     PostgresqlConnections `yaml:"connections"`
	Migrations      Migrations            `yaml:"migrations"`
	ConnectTimeout  Duration              `yaml:"connectTimeout"`
}

type PostgresqlConnections struct {
	MaxIdleConns    int      `yaml:"maxIdleConns"`
	MaxOpenConns    int      `yaml:"maxOpenConns"`
	ConnMaxLifetime Duration `yaml:"connMaxLifetime"`
	ConnMaxIdleTime Duration `yaml:"connMaxIdleTime"`
}

type Migrations struct {
	IsEnabled bool   `yaml:"isEnabled"`
	DirPath   string `yaml:"dirPath"`
}

// DSN безопасно собирает строку подключения к PostgreSQL из отдельных параметров
func (c PostgresqlInstance) DSN() string {
	value := &url.URL{
		Scheme: "postgres",
		User:   url.UserPassword(c.UserName, c.Password),
		Host:   net.JoinHostPort(c.Host, strconv.Itoa(c.Port)),
		Path:   c.DBName,
	}
	query := value.Query()
	query.Set("sslmode", c.SSLMode)
	if c.ApplicationName != "" {
		query.Set("application_name", c.ApplicationName)
	}
	value.RawQuery = query.Encode()
	return value.String()
}

type Application struct {
	InstanceID string     `yaml:"instanceID"`
	Simulation Simulation `yaml:"simulation"`
}

type Simulation struct {
	IsEnabled bool     `yaml:"isEnabled"`
	Delay     Duration `yaml:"delay"`
	ErrorRate float64  `yaml:"errorRate"`
}

// Load строго читает YAML-конфигурацию, применяет значения по умолчанию и проверяет параметры
func Load(path string) (Config, error) {
	// Открываем только явно переданный файл без чтения переменных окружения
	file, err := os.Open(path)
	if err != nil {
		return Config{}, fmt.Errorf("open config: %w", err)
	}
	defer func() { _ = file.Close() }()

	var cfg Config
	decoder := yaml.NewDecoder(file)
	// Запрещаем неизвестные поля, чтобы опечатки не оставались незамеченными
	decoder.KnownFields(true)
	if err := decoder.Decode(&cfg); err != nil {
		return Config{}, fmt.Errorf("decode config: %w", err)
	}
	// Проверяем, что файл содержит ровно один YAML-документ
	var extra any
	if err := decoder.Decode(&extra); !errors.Is(err, io.EOF) {
		if err == nil {
			return Config{}, errors.New("decode config: multiple YAML documents are not allowed")
		}
		return Config{}, fmt.Errorf("decode config: %w", err)
	}
	applyDefaults(&cfg)
	if err := cfg.Validate(); err != nil {
		return Config{}, fmt.Errorf("validate config: %w", err)
	}
	return cfg, nil
}

// applyDefaults заполняет необязательные параметры стандартными значениями
func applyDefaults(cfg *Config) {
	if cfg.Logging.Level == "" {
		cfg.Logging.Level = "INFO"
	}
	if cfg.Logging.Encoding == "" {
		cfg.Logging.Encoding = "json"
	}
	if cfg.Logging.RequestIDFieldName == "" {
		cfg.Logging.RequestIDFieldName = "x-request-id"
	}
}

// Validate проверяет обязательные поля, диапазоны и допустимые значения конфигурации
func (c Config) Validate() error {
	var problems []string
	if c.Mode != "dev" && c.Mode != "prod" {
		problems = append(problems, "mode must be dev or prod")
	}
	if strings.TrimSpace(c.Application.InstanceID) == "" {
		problems = append(problems, "application.instanceID is required")
	}
	if !c.Listen.HTTP.IsEnabled {
		problems = append(problems, "listen.http.isEnabled must be true")
	}
	if c.Listen.HTTP.Port < 1 || c.Listen.HTTP.Port > 65535 {
		problems = append(problems, "listen.http.port must be between 1 and 65535")
	}
	for name, value := range map[string]time.Duration{
		"listen.http.maxRequestProcessingTime": c.Listen.HTTP.MaxRequestProcessingTime.Duration,
		"listen.http.readHeaderTimeout":        c.Listen.HTTP.ReadHeaderTimeout.Duration,
		"listen.http.readTimeout":              c.Listen.HTTP.ReadTimeout.Duration,
		"listen.http.writeTimeout":             c.Listen.HTTP.WriteTimeout.Duration,
		"listen.http.idleTimeout":              c.Listen.HTTP.IdleTimeout.Duration,
		"listen.http.shutdownTimeout":          c.Listen.HTTP.ShutdownTimeout.Duration,
	} {
		if value <= 0 {
			problems = append(problems, name+" must be positive")
		}
	}
	database := c.Infrastructure.Postgresql.RW
	if !database.IsEnabled {
		problems = append(problems, "infrastructure.postgresql.rw.isEnabled must be true")
	}
	if strings.TrimSpace(database.Host) == "" {
		problems = append(problems, "infrastructure.postgresql.rw.host is required")
	}
	if database.Port < 1 || database.Port > 65535 {
		problems = append(problems, "infrastructure.postgresql.rw.port must be between 1 and 65535")
	}
	if database.DBName == "" || database.UserName == "" {
		problems = append(problems, "infrastructure.postgresql.rw.dbName and userName are required")
	}
	if database.SSLMode == "" {
		problems = append(problems, "infrastructure.postgresql.rw.sslMode is required")
	}
	if database.Connections.MaxOpenConns <= 0 {
		problems = append(problems, "infrastructure.postgresql.rw.connections.maxOpenConns must be positive")
	}
	if database.Connections.MaxIdleConns < 0 || database.Connections.MaxIdleConns > database.Connections.MaxOpenConns {
		problems = append(problems, "infrastructure.postgresql.rw.connections.maxIdleConns must be between 0 and maxOpenConns")
	}
	if database.Connections.ConnMaxLifetime.Duration <= 0 || database.Connections.ConnMaxIdleTime.Duration <= 0 || database.ConnectTimeout.Duration <= 0 {
		problems = append(problems, "PostgreSQL connection durations must be positive")
	}
	if database.Migrations.IsEnabled && strings.TrimSpace(database.Migrations.DirPath) == "" {
		problems = append(problems, "infrastructure.postgresql.rw.migrations.dirPath is required when migrations are enabled")
	}
	if c.Application.Simulation.Delay.Duration < 0 {
		problems = append(problems, "application.simulation.delay cannot be negative")
	}
	if c.Application.Simulation.ErrorRate < 0 || c.Application.Simulation.ErrorRate > 1 {
		problems = append(problems, "application.simulation.errorRate must be between 0 and 1")
	}
	switch strings.ToUpper(c.Logging.Level) {
	case "DEBUG", "INFO", "WARN", "ERROR":
	default:
		problems = append(problems, "logging.level must be DEBUG, INFO, WARN, or ERROR")
	}
	if c.Logging.Encoding != "json" {
		problems = append(problems, "logging.encoding must be json")
	}
	if strings.TrimSpace(c.Logging.ApplicationName) == "" {
		problems = append(problems, "logging.applicationName is required")
	}
	if len(problems) > 0 {
		return errors.New(strings.Join(problems, "; "))
	}
	return nil
}

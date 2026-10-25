package configs

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"
)

const validConfig = `
mode: dev
logging:
  level: INFO
  encoding: json
  development: true
  disableCaller: true
  disableStacktrace: true
  requestIDFieldName: x-request-id
  applicationName: adaptive-routing-backend
listen:
  http:
    isEnabled: true
    host: 127.0.0.1
    port: 8080
    maxRequestProcessingTime: 10s
    readHeaderTimeout: 5s
    readTimeout: 10s
    writeTimeout: 10s
    idleTimeout: 1m
    shutdownTimeout: 10s
infrastructure:
  postgresql:
    rw:
      isEnabled: true
      host: localhost
      port: 5432
      dbName: app
      userName: app
      password: app
      sslMode: disable
      applicationName: adaptive-routing-backend
      connections:
        maxIdleConns: 5
        maxOpenConns: 10
        connMaxLifetime: 30m
        connMaxIdleTime: 5m
      migrations:
        isEnabled: true
        dirPath: migrations
      connectTimeout: 5s
application:
  instanceID: backend-test
  simulation:
    isEnabled: true
    delay: 300ms
    errorRate: 0.25
`

// TestLoad проверяет успешную загрузку корректной конфигурации
func TestLoad(t *testing.T) {
	cfg, err := Load(writeConfig(t, validConfig))
	if err != nil {
		t.Fatalf("Load() error = %v", err)
	}
	if cfg.Application.InstanceID != "backend-test" || cfg.Application.Simulation.Delay.Duration != 300*time.Millisecond {
		t.Fatalf("Load() = %#v", cfg)
	}
	if !strings.Contains(cfg.Infrastructure.Postgresql.RW.DSN(), "application_name=adaptive-routing-backend") {
		t.Fatalf("DSN() = %q", cfg.Infrastructure.Postgresql.RW.DSN())
	}
}

// TestLoadRejectsUnknownField проверяет запрет неизвестных YAML-полей
func TestLoadRejectsUnknownField(t *testing.T) {
	_, err := Load(writeConfig(t, validConfig+"unknown: true\n"))
	if err == nil || !strings.Contains(err.Error(), "field unknown not found") {
		t.Fatalf("Load() error = %v", err)
	}
}

// TestValidateRejectsInvalidValues проверяет отклонение недопустимых значений
func TestValidateRejectsInvalidValues(t *testing.T) {
	config := strings.Replace(validConfig, "errorRate: 0.25", "errorRate: 1.5", 1)
	_, err := Load(writeConfig(t, config))
	if err == nil || !strings.Contains(err.Error(), "application.simulation.errorRate") {
		t.Fatalf("Load() error = %v", err)
	}
}

// writeConfig создаёт временный YAML-файл для теста
func writeConfig(t *testing.T, contents string) string {
	t.Helper()
	path := filepath.Join(t.TempDir(), "env.yaml")
	if err := os.WriteFile(path, []byte(contents), 0o600); err != nil {
		t.Fatalf("write config: %v", err)
	}
	return path
}

# Backend-сервис для исследования адаптивной маршрутизации

Небольшой HTTP-сервис на Go для воспроизводимых экспериментов с маршрутизацией запросов между несколькими репликами. Реплики не хранят локальное состояние, работают с общей PostgreSQL и сообщают свой идентификатор в заголовке `X-Instance-ID`. API Gateway намеренно не входит в проект.

## Возможности

- создание и чтение записей через PostgreSQL;
- health check приложения и базы данных;
- три независимо настраиваемые реплики в Docker Compose;
- управляемая задержка и вероятность искусственного `503`;
- JSON-логи через `log/slog`, request ID и graceful shutdown;
- строгая YAML-конфигурация и OpenAPI 3 в `docs/swagger.yaml`.

## Архитектура

Структура повторяет организацию эталонного API-сервиса: точка входа находится в `cmd/bin`, сборка зависимостей — в `internal/app`, а прикладные слои — в `internal/api`.

```text
cmd/bin/                   точка входа приложения
configs/                   типы, загрузка и проверка конфигурации
delivery/http/model/       внешние HTTP-модели
internal/app/              composition root и жизненный цикл
internal/api/models/       внутренние модели
internal/api/domain/       контракты и бизнес-сценарии
internal/api/delivery/     HTTP handlers, router и middleware
internal/api/repositories/ PostgreSQL-репозиторий
pkg/api/                   общие HTTP, logger и error-компоненты
migrations/                SQL-миграции up/down
deployments/envs/           конфигурации реплик
deployments/local/          локальный Docker Compose
docs/swagger.yaml           контракт HTTP API
```

## Конфигурация

Приложение читает все настройки только из YAML. Путь задаётся флагом `-config`; значение по умолчанию — `env.yaml`. Структура параметров совпадает со стилем эталонного сервиса. Неизвестные поля и некорректные значения останавливают запуск.

```yaml
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
    host: 0.0.0.0
    port: 8080
    maxRequestProcessingTime: 10s
    readHeaderTimeout: 5s
    readTimeout: 10s
    writeTimeout: 10s
    idleTimeout: 60s
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
  instanceID: backend-1
  simulation:
    isEnabled: false
    delay: 0s
    errorRate: 0
```

Полный рабочий пример находится в `env.yaml`. Длительности используют синтаксис Go (`300ms`, `5s`, `30m`). `application.simulation.errorRate` находится в диапазоне от `0` до `1`. Реквизиты `app:app` предназначены только для локальной демонстрационной базы и не являются production-секретами.

## Локальный запуск

Запустите PostgreSQL, затем сервис:

```bash
go run ./cmd/bin -config env.yaml

# Только проверить подключение к PostgreSQL
go run ./cmd/bin -config env.yaml -dry-run
```

При запуске сервис сам применяет все отсутствующие `up`-миграции из каталога `infrastructure.postgresql.rw.migrations.dirPath`. Значение `isEnabled: false` отключает автоматическое изменение схемы. Одновременный запуск нескольких реплик безопасен благодаря блокировке мигратора в PostgreSQL.

## Запуск трёх реплик

```bash
docker compose -f deployments/local/docker-compose.yml up --build
```

Compose запускает PostgreSQL и три реплики на портах `8081`, `8082`, `8083`. Каждая реплика проверяет и при необходимости применяет миграции перед запуском HTTP-сервера. Все реплики используют одну БД. Остановка:

```bash
docker compose -f deployments/local/docker-compose.yml down
```

Чтобы удалить также данные демонстрационной БД:

```bash
docker compose -f deployments/local/docker-compose.yml down -v
```

## Примеры API

Проверка состояния:

```bash
curl -i http://localhost:8081/health
```

Создание записи через первую реплику:

```bash
curl -i -X POST http://localhost:8081/api/v1/items \
  -H 'Content-Type: application/json' \
  -d '{"name":"test item","value":"test value"}'
```

Получение созданного UUID через другую реплику:

```bash
curl -i http://localhost:8082/api/v1/items/53dc68a4-458b-4d2f-b1c3-83ace8f4088d
```

Ошибки имеют единый формат:

```json
{"error":{"code":"validation_error","message":"field name is required"}}
```

## Имитация деградации

Механизм применяется только к `/api/v1`; `/health` никогда искусственно не задерживается и не завершается смоделированной ошибкой. Например, для третьей реплики измените `deployments/envs/backend-3.yaml`:

```yaml
application:
  instanceID: backend-3
  simulation:
    isEnabled: true
    delay: 50ms
    errorRate: 0.3
```

Пересоздайте только эту реплику:

```bash
docker compose -f deployments/local/docker-compose.yml up -d --build --force-recreate backend-3
```

Задержка прерывается при отмене HTTP-запроса. Случайное решение об ошибке безопасно при конкурентных запросах.

## Проверки

```bash
gofmt -w cmd configs delivery internal pkg
go test ./...
go test -race ./...
go vet ./...
docker run --rm -v "$PWD:/app" -w /app golangci/golangci-lint:v2.1.6 golangci-lint run
```

Тесты покрывают use case, строгую POST-валидацию, HTTP-преобразование not found, оба состояния health check, заголовок реплики и simulation middleware.

# PR Reviewer Assignment Service

Сервис для автоматического назначения ревьюеров на Pull Request'ы из команды автора.

## Описание

Сервис предоставляет HTTP API для управления командами, пользователями и автоматического назначения ревьюеров на Pull Request'ы. При создании PR автоматически назначаются до двух активных ревьюеров из команды автора.

## Архитектура

Проект использует Package Oriented Design (POD) с элементами Clean Architecture в формате модульного монолита. Подробнее см. [docs/ARCHITECTURE.md](docs/ARCHITECTURE.md).

## Быстрый старт

### Запуск через Docker Compose (рекомендуется)

Самый простой способ запустить сервис:

```bash
docker-compose up
```

Сервис будет доступен на порту 8080. PostgreSQL запустится автоматически, миграции применятся при старте приложения.

### Локальная разработка

1. Убедитесь, что установлены Go 1.22+ и PostgreSQL 12+

2. Установите зависимости:

```bash
go mod download
```

3. Настройте переменные окружения (см. раздел "Переменные окружения")

4. Запустите миграции и сервер:

```bash
# Миграции применяются автоматически при запуске
go run cmd/server/main.go
```

## Переменные окружения

### Сервер

- `SERVER_HOST` - хост сервера (по умолчанию: пустая строка - все интерфейсы)
- `SERVER_PORT` - порт сервера (по умолчанию: `:8080`)
- `SERVER_READ_TIMEOUT` - таймаут чтения запроса (по умолчанию: `10s`)
- `SERVER_WRITE_TIMEOUT` - таймаут записи ответа (по умолчанию: `10s`)
- `SERVER_IDLE_TIMEOUT` - таймаут простоя соединения (по умолчанию: `120s`)
- `GIN_MODE` - режим Gin (по умолчанию: `release`)

### База данных

- `DB_HOST` - хост PostgreSQL (по умолчанию: `localhost`)
- `DB_USER` - пользователь БД (по умолчанию: `postgres`)
- `DB_PASSWORD` - пароль БД (по умолчанию: `postgres`)
- `DB_NAME` - имя БД (по умолчанию: `avito_internship`)
- `DB_PORT` - порт PostgreSQL (по умолчанию: `5432`)
- `DB_SSLMODE` - режим SSL (по умолчанию: `disable`)
- `DB_TIMEZONE` - часовой пояс (по умолчанию: `UTC`)

### Логгер

- `LOGGER_LEVEL` - уровень логирования (по умолчанию: `info`)
- `LOGGER_FORMAT` - формат логов (`json` или `console`, по умолчанию: `json`)
- `LOGGER_OUTPUT` - вывод логов (`stdout` или `stderr`, по умолчанию: `stdout`)
- `LOGGER_PRODUCTION` - режим production (по умолчанию: `true`)

### Миграции

- `MIGRATIONS_PATH` - путь к директории с миграциями (по умолчанию: `migrations`)

## API

### Health Check

```bash
GET /health
```

Проверка состояния сервиса и подключения к БД.

### Teams

#### Создать команду

```bash
POST /team/add
Content-Type: application/json

{
  "team_name": "backend",
  "members": [
    {
      "user_id": "u1",
      "username": "Alice",
      "is_active": true
    },
    {
      "user_id": "u2",
      "username": "Bob",
      "is_active": true
    }
  ]
}
```

#### Получить команду

```bash
GET /team/get?team_name=backend
```

### Users

#### Установить активность пользователя

```bash
POST /users/setIsActive
Content-Type: application/json

{
  "user_id": "u1",
  "is_active": false
}
```

#### Получить PR'ы пользователя

```bash
GET /users/getReview?user_id=u1
```

### Pull Requests

#### Создать PR

```bash
POST /pullRequest/create
Content-Type: application/json

{
  "pull_request_id": "pr-1001",
  "pull_request_name": "Add search feature",
  "author_id": "u1"
}
```

Автоматически назначаются до 2 активных ревьюеров из команды автора.

#### Объединить PR

```bash
POST /pullRequest/merge
Content-Type: application/json

{
  "pull_request_id": "pr-1001"
}
```

Операция идемпотентна - повторный вызов не приводит к ошибке.

#### Переназначить ревьювера

```bash
POST /pullRequest/reassign
Content-Type: application/json

{
  "pull_request_id": "pr-1001",
  "old_user_id": "u2"
}
```

Новый ревьювер выбирается случайно из команды заменяемого ревьювера.

## Структура проекта

```
.
├── cmd/
│   └── server/          # Точка входа приложения
├── internal/
│   ├── config/          # Конфигурация
│   ├── database/        # Подключение к БД и миграции
│   ├── health/          # Health check endpoint
│   ├── middleware/      # HTTP middleware (logger, recovery)
│   ├── pullrequest/     # Модуль Pull Requests
│   ├── team/            # Модуль Teams
│   └── user/            # Модуль Users
├── migrations/          # SQL миграции
├── pkg/                 # Общие пакеты (logger, retry)
├── tests/               # E2E тесты
├── api/                 # OpenAPI спецификация
└── docs/                # Документация
```

## Тестирование

### Unit тесты

```bash
go test ./...
```

### Тесты с покрытием

```bash
make test-coverage
```

### E2E тесты

```bash
make test-e2e
```

## Линтинг

```bash
make lint
```

Автоисправление:

```bash
make lint-fix
```

## Требования

- Go 1.22+
- PostgreSQL 12+
- Docker и Docker Compose (для запуска через docker-compose)

## Документация

- [Архитектура](docs/ARCHITECTURE.md)
- [Развертывание](docs/DEPLOYMENT.md)
- [Контрибуция](docs/CONTRIBUTING.md)
- [Линтер](docs/LINTER.md)
- [OpenAPI спецификация](api/openapi.yml)

## Лицензия

См. файл [LICENSE](LICENSE).

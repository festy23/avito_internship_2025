# PR Reviewer Assignment Service

Сервис для автоматического назначения ревьюеров на Pull Request'ы из команды автора.

## Автор

Проект выполнил **Коновалов Иван**.

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

3. Настройте переменные окружения:

   Создайте файл `.env` в корне проекта (см. пример в разделе "Переменные окружения") или экспортируйте переменные окружения вручную.

4. Запустите PostgreSQL локально или используйте Docker:

   ```bash
   # Запуск только PostgreSQL через Docker
   docker-compose up postgres -d
   ```

5. Запустите сервер (миграции применяются автоматически при старте):

   ```bash
   go run cmd/server/main.go
   ```

   Сервис будет доступен на `http://localhost:8080`

## Переменные окружения

### Сервер

- `SERVER_HOST` - хост сервера (по умолчанию: пустая строка - все интерфейсы)
- `SERVER_PORT` - порт сервера (по умолчанию: `:8080`)
- `SERVER_READ_TIMEOUT` - таймаут чтения запроса (по умолчанию: `10s`)
- `SERVER_WRITE_TIMEOUT` - таймаут записи ответа (по умолчанию: `10s`)
- `SERVER_IDLE_TIMEOUT` - таймаут простоя соединения (по умолчанию: `120s`)
- `GIN_MODE` - режим Gin (`debug`, `release`, `test`, по умолчанию: `release`)

### База данных

- `DB_HOST` - хост PostgreSQL (по умолчанию: `localhost`, в Docker: `postgres`)
- `DB_USER` - пользователь БД (по умолчанию: `postgres`)
- `DB_PASSWORD` - пароль БД (по умолчанию: `postgres`)
- `DB_NAME` - имя БД (по умолчанию: `avito_internship`)
- `DB_PORT` - порт PostgreSQL (по умолчанию: `5432`)
- `DB_SSLMODE` - режим SSL (по умолчанию: `disable`)
- `DB_TIMEZONE` - часовой пояс (по умолчанию: `UTC`)

### Повторные попытки подключения к БД

- `DB_RETRY_MAX_ATTEMPTS` - максимальное количество попыток подключения (по умолчанию: `5`)
- `DB_RETRY_INITIAL_DELAY` - начальная задержка между попытками (по умолчанию: `1s`)
- `DB_RETRY_MAX_DELAY` - максимальная задержка между попытками (по умолчанию: `30s`)
- `DB_RETRY_MULTIPLIER` - множитель для экспоненциальной задержки (по умолчанию: `2.0`)

### Логгер

- `LOG_LEVEL` - уровень логирования (`debug`, `info`, `warn`, `error`, по умолчанию: `info`)
- `LOG_FORMAT` - формат логов (`json` или `console`, по умолчанию: `json`)
- `LOG_OUTPUT` - вывод логов (`stdout`, `stderr` или путь к файлу, по умолчанию: `stdout`)

### Миграции

- `MIGRATIONS_PATH` - путь к директории с миграциями (по умолчанию: `migrations`)

### Пример файла .env

Для локальной разработки создайте файл `.env` в корне проекта:

```bash
# Server
SERVER_HOST=
SERVER_PORT=:8080
SERVER_READ_TIMEOUT=10s
SERVER_WRITE_TIMEOUT=10s
SERVER_IDLE_TIMEOUT=120s
GIN_MODE=release

# Database
DB_HOST=localhost
DB_USER=postgres
DB_PASSWORD=postgres
DB_NAME=avito_internship
DB_PORT=5432
DB_SSLMODE=disable
DB_TIMEZONE=UTC

# Database Retry
DB_RETRY_MAX_ATTEMPTS=5
DB_RETRY_INITIAL_DELAY=1s
DB_RETRY_MAX_DELAY=30s
DB_RETRY_MULTIPLIER=2.0

# Logger
LOG_LEVEL=info
LOG_FORMAT=json
LOG_OUTPUT=stdout

# Migrations
MIGRATIONS_PATH=migrations
```

**Примечание:** При запуске через `docker-compose up` переменные окружения можно задать через файл `.env` в корне проекта или передать напрямую в `docker-compose.yml`.

## API

### Health Check

```bash
GET /health
```

Проверка состояния сервиса и подключения к БД.

**Ответ при успехе (200 OK):**

```json
{
  "status": "ok"
}
```

**Ответ при проблемах (503 Service Unavailable):**

```json
{
  "status": "unhealthy"
}
```

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

#### Массовая деактивация пользователей команды

```bash
POST /users/bulkDeactivate
Content-Type: application/json

{
  "team_name": "backend"
}
```

Деактивирует всех активных пользователей команды и автоматически переназначает ревьюверов в открытых PR на активных участников команды.

**Пример ответа:**

```json
{
  "team_name": "backend",
  "deactivated_users": ["u1", "u2"],
  "reassigned_prs": ["pr-1001", "pr-1002"],
  "deactivated_count": 2,
  "reassigned_pr_count": 2
}
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

### Statistics

#### Статистика по ревьюверам

```bash
GET /statistics/reviewers
```

Возвращает статистику по всем пользователям: количество назначений каждого ревьювера.

**Пример ответа:**

```json
{
  "reviewers": [
    {
      "user_id": "u1",
      "username": "Alice",
      "team_name": "backend",
      "assignment_count": 5,
      "is_active": true
    }
  ],
  "total": 1
}
```

#### Статистика по Pull Requests

```bash
GET /statistics/pullrequests
```

Возвращает статистику по PR: общее количество, распределение по статусам, среднее количество ревьюверов.

**Пример ответа:**

```json
{
  "statistics": {
    "total_prs": 10,
    "open_prs": 7,
    "merged_prs": 3,
    "average_reviewers_per_pr": 1.5,
    "prs_with_0_reviewers": 1,
    "prs_with_1_reviewer": 4,
    "prs_with_2_reviewers": 5
  }
}
```

## Структура проекта

```text
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

### Integration тесты

```bash
make test-integration
```

### E2E тесты

```bash
make test-e2e
```

Подробнее о тестировании см. [docs/TESTING.md](docs/TESTING.md).

### Load тесты

```bash
make test-load
```

Нагрузочное тестирование для проверки соответствия требованиям производительности (RPS 5, SLI 300 мс, SLI 99.9%).

Подробнее см. [docs/LOAD_TESTING.md](docs/LOAD_TESTING.md).

## Линтинг

```bash
make lint
```

Автоисправление:

```bash
make lint-fix
```

Подробнее о линтере см. [docs/LINTER.md](docs/LINTER.md).

## CI/CD

Проект использует GitHub Actions для автоматических проверок. CI/CD запускается автоматически при push и создании Pull Request.

### Локальная проверка CI/CD

Перед созданием PR рекомендуется запустить те же проверки локально:

```bash
make ci
```

Эта команда выполняет:

- Линтинг кода
- Integration тесты
- Unit тесты

### Локальное тестирование CI/CD

Для быстрой локальной проверки используйте команды `ci-local*` (запускают команды напрямую без Docker):

```bash
# Запуск всех CI checks
make ci-local

# Запуск отдельных checks
make ci-local-lint    # Только lint
make ci-local-test    # Только unit и integration тесты
make ci-local-e2e     # Только E2E тесты
```

### Локальное тестирование GitHub Actions

Для локального тестирования GitHub Actions workflows используется инструмент [act](https://github.com/nektos/act).

**Требования:**

- Docker должен быть установлен и запущен
- `act` установлен (через `brew install act` на macOS)

**Использование:**

```bash
# Просмотр доступных jobs
make ci-act-list

# Запуск всех CI jobs через act
make ci-act

# Запуск отдельных jobs через act
make ci-act-lint    # Только lint через act
make ci-act-test    # Только unit и integration тесты через act
make ci-act-e2e     # Только E2E тесты через act
```

Подробнее см. [docs/CI_LOCAL_TESTING.md](docs/CI_LOCAL_TESTING.md).

### GitHub Actions

Workflow файл: `.github/workflows/ci.yml`

Проверки:

- **Lint** - проверка кода линтером (всегда)
- **Test** - запуск unit тестов, генерация coverage report (всегда)
- **Integration Tests** - запуск integration тестов (только на `dev` и `main`)
- **E2E Tests** - запуск E2E тестов (только на `main`)

Статус проверок отображается в GitHub при создании Pull Request.

## Дополнительные задания

Все дополнительные задания из технического задания выполнены:

**Эндпоинт статистики** - реализованы эндпоинты `/statistics/reviewers` и `/statistics/pullrequests` для получения статистики по назначениям ревьюверов и PR

**Нагрузочное тестирование** - реализованы load тесты для проверки соответствия требованиям (RPS 5, SLI 300 мс, SLI 99.9%). Подробнее см. [docs/LOAD_TESTING.md](docs/LOAD_TESTING.md)

**Массовая деактивация пользователей** - реализован эндпоинт `/users/bulkDeactivate` для деактивации всех пользователей команды с автоматическим переназначением ревьюверов в открытых PR (оптимизировано для выполнения в пределах 100 мс)

**E2E тестирование** - реализованы комплексные E2E тесты, покрывающие основные бизнес-сценарии, граничные случаи и обработку ошибок

**Конфигурация линтера** - настроен golangci-lint с подробной документацией. Подробнее см. [docs/LINTER.md](docs/LINTER.md)

## Требования

- Go 1.22+
- PostgreSQL 12+
- Docker и Docker Compose (для запуска через docker-compose)

## Документация

- [Архитектура](docs/ARCHITECTURE.md)
- [Развертывание](docs/DEPLOYMENT.md)
- [Контрибуция](docs/CONTRIBUTING.md)
- [Тестирование](docs/TESTING.md)
- [Нагрузочное тестирование](docs/LOAD_TESTING.md)
- [Линтер](docs/LINTER.md)
- [Локальное тестирование CI](docs/CI_LOCAL_TESTING.md)
- [OpenAPI спецификация](api/openapi.yml)

### Примечание об OpenAPI спецификации

В файле `api/openapi.yml` обнаружена опечатка в примере для эндпоинта `/pullRequest/reassign` (строка 347): в примере используется `old_reviewer_id`, тогда как в схеме запроса (строка 344) корректно указано `old_user_id`.

**Важно:** Согласно техническому заданию, файл `openapi.yml` изменять нельзя, поэтому опечатка сохранена в спецификации. Реализация сервиса использует корректное поле `old_user_id`, что соответствует схеме запроса.

## AI Инструменты разработки

При разработке проекта использовались следующие AI-инструменты:

- **Perplexity** - для исследования технологий, архитектурных решений и best practices
- **Cursor** - для разработки и написания кода + документации
- **CodeRabbit** - для ревью Pull Request'ов

## Лицензия

См. файл [LICENSE](LICENSE).

# PR Reviewer Assignment Service

Сервис для автоматического назначения ревьюеров на Pull Request'ы из команды автора. Предоставляет HTTP API для управления командами, пользователями и автоматического назначения ревьюеров на PR.

## Быстрый старт

### Docker Compose

```bash
docker-compose up
```

Сервис доступен на порту 8080. PostgreSQL запускается автоматически, миграции применяются при старте.

Проверка работоспособности:

```bash
curl http://localhost:8080/health
```

### Локальная разработка

1. Установите зависимости: `go mod download`
2. Настройте переменные окружения (см. раздел "Переменные окружения")
3. Запустите PostgreSQL: `docker-compose up postgres -d`
4. Запустите сервер: `go run cmd/server/main.go`

## Требования

- Go 1.25.4+
- PostgreSQL 12+
- Docker и Docker Compose

## Документация

- [Архитектура](docs/ARCHITECTURE.md) - описание архитектуры проекта
- [Развертывание](docs/DEPLOYMENT.md) - руководство по развертыванию
- [Тестирование](docs/TESTING.md) - стратегия тестирования
- [Контрибуция](docs/CONTRIBUTING.md) - руководство по контрибуции
- [Нагрузочное тестирование](docs/LOAD_TESTING.md) - результаты нагрузочного тестирования
- [Линтер](docs/LINTER.md) - конфигурация линтера
- [Локальное тестирование CI](docs/CI_LOCAL_TESTING.md) - локальное тестирование CI/CD
- [Postman](docs/POSTMAN.md) - описание Postman коллекции
- [OpenAPI спецификация](api/openapi.yml) - полная спецификация API

## Описание

Сервис автоматически назначает ревьюеров на Pull Request'ы из команды автора. При создании PR автоматически назначаются до двух активных ревьюеров из команды автора (исключая самого автора).

Основные возможности:

- Автоматическое назначение до 2 ревьюеров при создании PR
- Переназначение ревьюверов в открытых PR
- Идемпотентная операция merge
- Массовая деактивация пользователей команды с автоматическим переназначением ревьюверов
- Статистика по ревьюверам и PR
- Health check endpoint

## Архитектура

Проект использует Package Oriented Design (POD) с элементами Clean Architecture в формате модульного монолита.

Архитектурные решения:

- Модульность по доменам (team, user, pullrequest, statistics)
- Трехслойная архитектура: handler → service → repository
- PostgreSQL 12 для хранения данных
- Gin framework для HTTP API
- GORM для работы с БД

Подробнее об архитектуре см. [docs/ARCHITECTURE.md](docs/ARCHITECTURE.md).

## API

Полная спецификация API доступна в [api/openapi.yml](api/openapi.yml).

### Примечание об OpenAPI спецификации

В файле `api/openapi.yml` обнаружена опечатка в примере для эндпоинта `/pullRequest/reassign` (строка 347): в примере используется `old_reviewer_id`, тогда как в схеме запроса (строка 344) корректно указано `old_user_id`.

**Важно:** Согласно техническому заданию, файл `openapi.yml` изменять нельзя, поэтому опечатка сохранена в спецификации. Реализация сервиса использует корректное поле `old_user_id`, что соответствует схеме запроса.

### Основные эндпоинты

**Teams:**

- `POST /team/add` - создать команду
- `GET /team/get?team_name=<name>` - получить команду

**Users:**

- `POST /users/setIsActive` - установить активность пользователя
- `GET /users/getReview?user_id=<id>` - получить PR'ы пользователя
- `POST /users/bulkDeactivate` - массовая деактивация пользователей команды

**Pull Requests:**

- `POST /pullRequest/create` - создать PR (автоназначение ревьюверов)
- `POST /pullRequest/merge` - объединить PR (идемпотентно)
- `POST /pullRequest/reassign` - переназначить ревьювера

**Statistics:**

- `GET /statistics/reviewers` - статистика по ревьюверам
- `GET /statistics/pullrequests` - статистика по PR

**Health:**

- `GET /health` - проверка состояния сервиса

## Переменные окружения

### Сервер

- `SERVER_HOST` - хост сервера (по умолчанию: `""`)
- `SERVER_PORT` - порт сервера (по умолчанию: `:8080`)
- `SERVER_READ_TIMEOUT` - таймаут чтения (по умолчанию: `10s`)
- `SERVER_WRITE_TIMEOUT` - таймаут записи (по умолчанию: `10s`)
- `SERVER_IDLE_TIMEOUT` - таймаут простоя (по умолчанию: `120s`)
- `GIN_MODE` - режим Gin (по умолчанию: `release`)

### База данных

- `DB_HOST` - хост PostgreSQL (по умолчанию: `localhost`, в Docker: `postgres`)
- `DB_USER` - пользователь БД (по умолчанию: `postgres`)
- `DB_PASSWORD` - пароль БД (по умолчанию: `postgres`)
- `DB_NAME` - имя БД (по умолчанию: `avito_internship`)
- `DB_PORT` - порт PostgreSQL (по умолчанию: `5432`)
- `DB_SSLMODE` - режим SSL (по умолчанию: `disable`)
- `DB_TIMEZONE` - часовой пояс (по умолчанию: `UTC`)

### Логгер

- `LOG_LEVEL` - уровень логирования (по умолчанию: `info`)
- `LOG_FORMAT` - формат логов (`json` или `console`, по умолчанию: `json`)
- `LOG_OUTPUT` - вывод логов (по умолчанию: `stdout`)

Полный список переменных окружения см. [docs/DEPLOYMENT.md](docs/DEPLOYMENT.md).

## Структура проекта

```text
.
├── cmd/server/          # Точка входа
├── internal/            # Внутренние модули
│   ├── config/         # Конфигурация
│   ├── database/        # Подключение к БД
│   ├── health/         # Health check
│   ├── middleware/     # HTTP middleware
│   ├── pullrequest/    # Модуль PR
│   ├── statistics/     # Модуль статистики
│   ├── team/           # Модуль команд
│   └── user/           # Модуль пользователей
├── migrations/         # SQL миграции
├── pkg/                # Общие пакеты
├── tests/              # Тесты (e2e, integration, load)
├── api/                # OpenAPI спецификация
└── docs/               # Документация
```

## Тестирование

Проект использует три уровня тестирования:

- **Unit тесты:** `go test ./...` или `make test`
- **Integration тесты:** `make test-integration`
- **E2E тесты:** `make test-e2e` (требует Docker)

Тесты с покрытием: `make test-coverage`

Подробнее см. [docs/TESTING.md](docs/TESTING.md).

## Развертывание

```bash
docker-compose up -d
```

Подробнее см. [docs/DEPLOYMENT.md](docs/DEPLOYMENT.md).

## Дополнительные задания

Все дополнительные задания выполнены:

- Эндпоинт статистики
- Нагрузочное тестирование (RPS 5, SLI 300 мс, SLI 99.9%)
- Массовая деактивация пользователей (оптимизировано для 100 мс)
- E2E тестирование
- Конфигурация линтера

## CI/CD

Проект использует GitHub Actions для автоматических проверок.

Локальная проверка: `make ci`

Подробнее см. [docs/CI_LOCAL_TESTING.md](docs/CI_LOCAL_TESTING.md).

## Линтинг

```bash
make lint        # Проверка
make lint-fix    # Автоисправление
```

Подробнее см. [docs/LINTER.md](docs/LINTER.md).

## Автор

Коновалов Иван

Проект выполнен в рамках тестового задания для стажёра Backend (осенняя волна 2025) в Авито.

## Лицензия

См. файл [LICENSE](LICENSE).

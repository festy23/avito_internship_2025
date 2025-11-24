# Документация по тестированию

## Обзор

Проект использует три уровня тестирования: unit тесты, integration тесты и E2E тесты.

## Unit тесты

### Расположение

- `internal/*/handler/handler_test.go` - тесты HTTP handlers
- `internal/*/service/service_test.go` - тесты бизнес-логики
- `internal/*/repository/repository_test.go` - тесты доступа к данным
- `internal/*/model/*_test.go` - тесты моделей данных

### Технологии

- `testify/mock` - для мокирования зависимостей
- `testify/assert` и `testify/require` - для проверок
- SQLite in-memory - для repository тестов
- `httptest` - для тестирования HTTP handlers

### Что проверяют

- Handler: HTTP запросы/ответы, валидация, маппинг ошибок
- Service: бизнес-логика с моками repository, правила назначения ревьюеров
- Repository: CRUD операции, работа с БД через GORM
- Model: JSON сериализация, валидация, GORM интеграция

### Запуск

```bash
go test ./...
# или
make test
```

## Integration тесты

### Расположение integration тестов

`tests/integration/`

### Build tag integration тестов

`integration`

### Технологии integration тестов

- SQLite in-memory
- `httptest.ResponseRecorder`
- Полный HTTP стек (Gin router, middleware, handlers)
- AutoMigrate

### Что проверяют integration тесты

- Полный жизненный цикл PR (создание, автоназначение, мерж, переприсвоение)
- Управление командами и пользователями
- Обработка ошибок и граничные случаи

### Запуск integration тестов

```bash
make test-integration
# или
go test -tags=integration ./tests/integration/... -v
```

## E2E тесты

### Расположение E2E тестов

`tests/e2e/`

### Build tag E2E тестов

`e2e`

### Технологии E2E тестов

- `testcontainers-go` для Docker контейнеров
- PostgreSQL 12 (реальная БД)
- Реальный HTTP сервер
- Реальные миграции

### Что проверяют E2E тесты

- Business Scenarios: полный жизненный цикл PR, управление активностью
- Error Scenarios: обработка ошибок (`NO_CANDIDATE`, `NOT_ASSIGNED`)
- Advanced Scenarios: конкурентность, идемпотентность, дублирование ключей
- Edge Cases: Unicode символы, длинные имена, пустые списки

### Запуск E2E тестов

```bash
make test-e2e
# или
go test -tags=e2e ./tests/e2e/... -v -timeout 20m
```

**Требования:** Docker daemon должен быть запущен.

## Сравнительная таблица

| Аспект | Unit тесты | Integration тесты | E2E тесты |
|--------|-----------|------------------|-----------|
| Расположение | `internal/*/...` | `tests/integration/` | `tests/e2e/` |
| Build tag | нет | `integration` | `e2e` |
| База данных | SQLite in-memory / моки | SQLite in-memory | PostgreSQL 12 (Docker) |
| HTTP | httptest / моки | httptest | Реальный HTTP сервер |
| Скорость | Очень быстро | Быстро | Медленно |
| Зависимости | Нет | Нет | Docker required |
| Миграции | AutoMigrate | AutoMigrate | Реальные миграции |

## Команды для запуска

### Команды для unit тестов

```bash
make test
# или
go test ./...
```

### Команды для integration тестов

```bash
make test-integration
```

### Команды для E2E тестов

```bash
make test-e2e
```

### С покрытием кода

```bash
make test-coverage
```

Генерирует отчет в `coverage.html` и `coverage.out`.

## CI/CD

Проект использует GitHub Actions для автоматического запуска тестов.

### Локальная проверка

```bash
make ci
```

Выполняет: линтинг кода, integration тесты, unit тесты.

### GitHub Actions

Workflow файл: `.github/workflows/ci.yml`

Jobs:

- `lint` - проверка кода линтером
- `test` - запуск unit и integration тестов, генерация coverage report

Триггеры: Push в ветки `main` и `dev`, Pull Request в ветки `main` и `dev`.

## Покрытие кода

### Генерация отчета

```bash
make test-coverage
```

### Целевое покрытие

- Бизнес-логика (service): ≥ 80%
- Handlers: ≥ 70%
- Repository: ≥ 70%
- Models: ≥ 60%

### Просмотр покрытия

```bash
open coverage.html
```

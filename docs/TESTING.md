# Testing Documentation

## Обзор

Проект использует три уровня тестирования: unit тесты, integration тесты и E2E тесты. Все тесты проверяют бизнес-логику через API, а не через прямые запросы к базе данных.

---

## Unit тесты

**Расположение**: `internal/*/handler/handler_test.go`, `internal/*/service/service_test.go`, `internal/*/repository/repository_test.go`, `internal/*/model/*_test.go`

**Технологии**: 
- `testify/mock` для моков зависимостей
- SQLite in-memory для repository тестов
- `testify/assert` и `testify/require` для проверок

**Цель**: Проверка изолированных компонентов без внешних зависимостей.

**Что проверяют**:
- **Handler**: HTTP запросы/ответы, валидация, маппинг ошибок на HTTP коды
- **Service**: Бизнес-логика с моками repository, правила назначения ревьюеров, транзакции
- **Repository**: CRUD операции, работа с БД через GORM, ограничения БД
- **Model**: JSON сериализация, валидация, GORM интеграция, доменные ошибки

**Запуск**:
```bash
go test ./internal/... -v
```

---

## Integration тесты

**Расположение**: `tests/integration/`

**Build tag**: `integration`

**Технологии**:
- SQLite in-memory
- `httptest.ResponseRecorder`
- Полный HTTP стек

**Цель**: Быстрые тесты бизнес-логики и API контрактов без внешних зависимостей.

**Характеристики**:
- Быстрое выполнение (секунды)
- Не требуют Docker
- Используют AutoMigrate вместо реальных миграций
- Подходят для CI/CD

**Что проверяют**:
- Полный жизненный цикл PR (создание, автоназначение ревьюеров, мерж, переприсвоение)
- Управление командами (создание, получение, множественные команды)
- Управление пользователями (активность, получение списка PR для ревьюера)
- Обработка ошибок и граничные случаи

**Запуск**:
```bash
make test-integration
# или
go test -tags=integration ./tests/integration/... -v
```

---

## E2E тесты

**Расположение**: `tests/e2e/`

**Build tag**: `e2e`

**Технологии**:
- `testcontainers-go` для Docker контейнеров
- PostgreSQL 12 (реальная БД)
- Реальный HTTP сервер
- Полный стек приложения

**Цель**: Проверка всей системы в условиях, близких к продакшену.

**Характеристики**:
- Медленное выполнение (минуты)
- Требуют Docker daemon
- Используют реальные миграции
- Проверяют PostgreSQL constraints, triggers, indexes
- Проверяют конкурентность и race conditions

**Что проверяют**:

**Business Scenarios** (`business_scenarios_test.go`):
- Полный жизненный цикл PR (создание → автоназначение → переприсвоение → мерж → идемпотентность)
- Управление активностью пользователей (неактивные не назначаются, деактивация/реактивация)
- Лимиты количества ревьюеров (0 для 1 члена, 1 для 2 членов, 2 для 3+ членов)

**Error Scenarios** (`error_scenarios_test.go`):
- Ошибка `NO_CANDIDATE` при переприсвоении
- Ошибка `NOT_ASSIGNED` при попытке переприсвоить не назначенного ревьюера
- Множественные PR и `getReview` (возврат всех PR, включая MERGED)

**Advanced Scenarios** (`advanced_scenarios_test.go`):
- Конкурентное создание PR (race conditions, справедливое распределение)
- Идемпотентность мержа (повторный мерж не меняет состояние)
- Дублирование ключей (`TEAM_EXISTS`, `PR_EXISTS`)
- Ошибки `NOT_FOUND` для всех endpoints

**Edge Cases** (`edge_cases_test.go`):
- Unicode и специальные символы (кириллица, японские, китайские символы)
- Цепочки переприсвоений
- Неизменяемость команд
- Длинные имена (лимит 255 символов)
- Пустые списки ревьюеров

**Запуск**:
```bash
make test-e2e
# или
go test -tags=e2e ./tests/e2e/... -v -timeout 20m
```

**Требования**:
- Docker daemon должен быть запущен
- Достаточно ресурсов для Docker контейнеров

---

## Сравнительная таблица

| Аспект | Unit тесты | Integration тесты | E2E тесты |
|--------|-----------|------------------|-----------|
| **Расположение** | `internal/*/...` | `tests/integration/` | `tests/e2e/` |
| **Build tag** | нет | `integration` | `e2e` |
| **База данных** | SQLite in-memory / моки | SQLite in-memory | PostgreSQL 12 (Docker) |
| **HTTP** | httptest / моки | httptest | Реальный HTTP сервер |
| **Скорость** | Очень быстро | Быстро | Медленно (минуты) |
| **Зависимости** | Нет | Нет | Docker required |
| **Миграции** | AutoMigrate | AutoMigrate | Реальные миграции |
| **Использование** | При разработке | CI/CD, быстрая обратная связь | Pre-release валидация |

---

## Команды для запуска

### Все unit тесты

```bash
make test
# или
go test ./...
```

### Integration тесты

```bash
make test-integration
# или
go test -tags=integration ./tests/integration/... -v
``

### E2E тесты

```bash
make test-e2e
# или
go test -tags=e2e ./tests/e2e/... -v -timeout 20m
```

### С покрытием кода

```bash
make test-coverage
```

## CI/CD

Проект использует GitHub Actions для автоматического запуска тестов при push и создании Pull Request.

### Локальная проверка CI/CD

Перед созданием PR рекомендуется запустить те же проверки, что выполняются в CI/CD:

```bash
make ci
```

Эта команда выполняет:

- Линтинг кода (`make lint`)
- Integration тесты (`make test-integration`)
- Unit тесты (`make test`)

### GitHub Actions

Workflow файл: `.github/workflows/ci.yml`

**Jobs:**

- `lint` - проверка кода линтером (golangci-lint)
- `test` - запуск unit и integration тестов, генерация coverage report

**Триггеры:**

- Push в ветки `main` и `dev`
- Pull Request в ветки `main` и `dev`

**Требования:**

- Все проверки должны пройти успешно для мержа PR
- Coverage report загружается как артефакт

### Запуск тестов в CI/CD

В CI/CD выполняются:

- Integration тесты с тегом `integration`
- Unit тесты для всех модулей
- Генерация coverage report

E2E тесты не запускаются в CI/CD по умолчанию (требуют Docker и больше времени), но могут быть добавлены при необходимости.

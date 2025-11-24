# Руководство по контрибуции

## Git Workflow

### Роли веток

- `main` - релизная ветка, стабильный код
- `dev` - ветка разработки, интеграция изменений
- Остальные ветки временные и удаляются после мержа

### Именование веток

- `feature/<модуль>/<описание>` - фичи
- `hotfix/<модуль>/<описание>` - срочные исправления
- `exp/<тема>` - эксперименты
- `support/<тема>` - документация, инфраструктура

### Процесс работы

1. Создать ветку от `dev`: `git checkout -b feature/user/add-status`
2. Внести изменения и закоммитить
3. Синхронизировать с `dev` (если нужно): `git rebase dev`
4. Запушить ветку: `git push origin feature/user/add-status`
5. Создать Pull Request в `dev`
6. После проверок CI/CD и ревью - мерж

## Commits

### Формат

Используйте Conventional Commits:

```text
<type>(<scope>): <subject>

<body>

<footer>
```

### Типы

- `feat` - новая функциональность
- `fix` - исправление бага
- `docs` - документация
- `refactor` - рефакторинг
- `test` - тесты
- `chore` - конфигурация, зависимости

### Scope

Модуль или компонент:

- `team`, `user`, `pullrequest`, `config`, `database`, `api`, `migration`

### Правила коммитов

- Subject: до 50 символов, маленькая буква, без точки
- Body: опционально, до 72 символов в строке
- Footer: опционально, ссылки на issues (`Closes #123`)

### Примеры

```text
feat(user): add method to get user activity

fix(pullrequest): fix reviewer assignment logic

docs(api): update OpenAPI examples
```

## Pull Requests

### Название

Формат как у коммитов: `<type>(<scope>): <описание>`

Примеры:

- `feat(pullrequest): add automatic picking reviewer`
- `fix(user): fix validation activity`

### Описание

Обязательно укажите:

- Что изменено
- Почему
- Как протестировать

### Правила

- Один PR = одна логическая фича
- До 400-500 строк изменений
- Линтер без ошибок
- Тесты для новой функциональности
- Комментарии на английском

### CI/CD

Проект использует GitHub Actions для автоматических проверок.

Проверки:

- Lint - проверка кода линтером
- Test - запуск unit и integration тестов

Локальная проверка: `make ci`

### Merge

- Squash and merge для feature-веток
- Удалить ветку после мержа

## Код-стайл

### Форматирование

```bash
go fmt ./...
```

### Линтинг

```bash
make lint        # Проверка
make lint-fix    # Автоисправление
```

### Комментарии

- Комментарии на английском языке
- Комментарии должны объяснять "почему", а не "что"
- Используйте `godoc` формат для публичных функций

## Тестирование

### Unit тесты

Добавляйте unit тесты для новой функциональности:

```bash
go test ./internal/user/service/... -v
```

### Integration тесты

```bash
make test-integration
```

### E2E тесты

```bash
make test-e2e
```

### Покрытие кода

```bash
make test-coverage
```

Стремитесь к покрытию не менее 70% для бизнес-логики.

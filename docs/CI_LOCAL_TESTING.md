# Локальное тестирование CI

Для локального тестирования GitHub Actions workflows используется инструмент [act](https://github.com/nektos/act).

## Установка

### macOS (через Homebrew)

```bash
brew install act
```

### Linux

```bash
curl https://raw.githubusercontent.com/nektos/act/master/install.sh | sudo bash
```

### Windows

```bash
choco install act-cli
```

## Требования

- Docker должен быть установлен и запущен
- Go 1.25.4+ (для тестов)

## Использование

### Локальные команды (без act)

Команды `ci-local*` запускают проверки напрямую на вашей машине без Docker контейнеров — это быстрее и надёжнее:

```bash
# Запуск всех CI checks
make ci-local

# Запуск отдельных checks
make ci-local-lint    # Только lint
make ci-local-test    # Только unit и integration тесты
make ci-local-e2e     # Только E2E тесты
```

### Команды через act (эмуляция GitHub Actions)

Команды `ci-act*` используют act для полной эмуляции GitHub Actions workflows:

> **Важно:** Убедитесь, что Docker Desktop выделено минимум 4 ГБ RAM и доступен сокет `/var/run/docker.sock` (требуется для шага Buildx в E2E job).

```bash
# Просмотр доступных jobs
make ci-act-list
# или
act --list -W .github/workflows/ci.yml

# Запуск всех CI jobs через act
make ci-act

# Запуск отдельных jobs через act
make ci-act-lint    # Только lint через act
make ci-act-test    # Только unit и integration тесты через act
make ci-act-e2e     # Только E2E тесты через act
```

**Примечание:** Все команды `ci-act-*` автоматически используют fallback на прямые команды (`make lint`, `make test`, и т.д.), если `act` не работает или завершается с ошибкой.

### Очистка контейнеров act

Если возникают проблемы с Docker контейнерами:

```bash
make ci-local-clean
```

Эта команда удалит все контейнеры и образы, связанные с act.

### Прямое использование act

```bash
# Запустить конкретный job с автоматической очисткой контейнеров
act -j lint -W .github/workflows/ci.yml --container-architecture linux/amd64 --rm

# Запустить с конкретным событием
act push -W .github/workflows/ci.yml --container-architecture linux/amd64 --rm

# Запустить с verbose выводом
act -j test -W .github/workflows/ci.yml -v

# Запустить с использованием secrets (если нужно)
act -j test -W .github/workflows/ci.yml --secret-file .secrets
```

## Особенности для macOS (Apple Silicon)

Если вы используете Mac с Apple Silicon (M1/M2/M3), может потребоваться указать архитектуру контейнера:

```bash
act -j test -W .github/workflows/ci.yml --container-architecture linux/amd64
```

## Ограничения

- `act` не полностью эмулирует GitHub Actions, некоторые функции могут работать по-другому
- Secrets нужно передавать явно через `--secret` или `--secret-file`
- Некоторые actions могут требовать дополнительной настройки
- E2E тесты могут работать медленнее локально из-за эмуляции

## Troubleshooting

### Docker не запущен

```bash
# Проверить статус Docker
docker ps

# Запустить Docker Desktop (macOS/Windows)
# или
sudo systemctl start docker  # Linux
```

### Ошибка "RWLayer of container is unexpectedly nil"

Эта ошибка возникает из-за поврежденных Docker контейнеров от предыдущих запусков act.

**Решение:**

```bash
# Очистить контейнеры act
make ci-local-clean

# Или вручную
docker ps -a --filter "ancestor=catthehacker/ubuntu:act-latest" --format "{{.ID}}" | xargs docker rm -f
docker system prune -f --filter "label=com.github.actions"
```

После очистки попробуйте запустить команду снова. Все команды `ci-act-*` используют флаг `--rm` для автоматической очистки контейнеров после выполнения.

### Проблемы с архитектурой на Apple Silicon

```bash
# Использовать платформу linux/amd64 (уже добавлено в команды)
act -j test -W .github/workflows/ci.yml --container-architecture linux/amd64 --rm
```

### Проблемы с правами доступа

```bash
# Убедитесь, что пользователь в группе docker
sudo usermod -aG docker $USER
```

### Если act не работает

Команды `ci-act-*` автоматически переключатся на прямые команды при сбое act:
- `make ci-act-lint` → `make lint`
- `make ci-act-test` → `make test-integration test`
- `make ci-act-e2e` → `make test-e2e`

Если act не установлен или не работает, используйте команды `ci-local-*` для локальной проверки без Docker.

## Полезные ссылки

- [act GitHub](https://github.com/nektos/act)
- [act Documentation](https://github.com/nektos/act#example-commands)

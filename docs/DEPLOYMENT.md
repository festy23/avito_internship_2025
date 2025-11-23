# Deployment Guide

Руководство по развертыванию PR Reviewer Assignment Service.

## Требования

- Go 1.25.4+
- PostgreSQL 12+
- Docker и Docker Compose (для контейнеризованного развертывания)

## Развертывание через Docker Compose

Самый простой способ развернуть сервис:

```bash
docker-compose up -d
```

Сервис будет доступен на порту 8080. PostgreSQL запустится автоматически, миграции применятся при старте приложения.

### Проверка работоспособности

```bash
curl http://localhost:8080/health
```

Ожидаемый ответ:
```json
{
  "status": "healthy",
  "database": "ok",
  "timestamp": "2025-11-23T20:00:00Z"
}
```

## Локальное развертывание

### 1. Установка зависимостей

```bash
go mod download
```

### 2. Настройка базы данных

Создайте базу данных PostgreSQL:

```sql
CREATE DATABASE avito_internship;
CREATE USER postgres WITH PASSWORD 'postgres';
GRANT ALL PRIVILEGES ON DATABASE avito_internship TO postgres;
```

### 3. Настройка переменных окружения

Создайте файл `.env` или экспортируйте переменные:

```bash
export DB_HOST=localhost
export DB_USER=postgres
export DB_PASSWORD=postgres
export DB_NAME=avito_internship
export DB_PORT=5432
export DB_SSLMODE=disable
export SERVER_PORT=:8080
export GIN_MODE=release
```

### 4. Применение миграций

Миграции применяются автоматически при запуске приложения. Если нужно применить вручную:

```bash
# Миграции находятся в директории migrations/
# Применяются автоматически через migrate.Up() при старте
```

### 5. Запуск сервера

```bash
go run cmd/server/main.go
```

Или скомпилируйте и запустите:

```bash
go build -o server cmd/server/main.go
./server
```

## Переменные окружения

### Сервер

| Переменная | Описание | По умолчанию |
|-----------|----------|--------------|
| `SERVER_HOST` | Хост сервера | `""` (все интерфейсы) |
| `SERVER_PORT` | Порт сервера | `:8080` |
| `SERVER_READ_TIMEOUT` | Таймаут чтения запроса | `10s` |
| `SERVER_WRITE_TIMEOUT` | Таймаут записи ответа | `10s` |
| `SERVER_IDLE_TIMEOUT` | Таймаут простоя соединения | `120s` |
| `GIN_MODE` | Режим Gin | `release` |

### База данных

| Переменная | Описание | По умолчанию |
|-----------|----------|--------------|
| `DB_HOST` | Хост PostgreSQL | `localhost` |
| `DB_USER` | Пользователь БД | `postgres` |
| `DB_PASSWORD` | Пароль БД | `postgres` |
| `DB_NAME` | Имя БД | `avito_internship` |
| `DB_PORT` | Порт PostgreSQL | `5432` |
| `DB_SSLMODE` | Режим SSL | `disable` |
| `DB_TIMEZONE` | Часовой пояс | `UTC` |
| `DB_RETRY_MAX_ATTEMPTS` | Максимум попыток подключения | `5` |
| `DB_RETRY_INITIAL_DELAY` | Начальная задержка между попытками | `1s` |
| `DB_RETRY_MAX_DELAY` | Максимальная задержка | `30s` |
| `DB_RETRY_MULTIPLIER` | Множитель задержки | `2.0` |

### Логгер

| Переменная | Описание | По умолчанию |
|-----------|----------|--------------|
| `LOG_LEVEL` | Уровень логирования | `info` |
| `LOG_FORMAT` | Формат логов (`json` или `console`) | `json` |
| `LOG_OUTPUT` | Вывод логов (`stdout` или `stderr`) | `stdout` |

### Миграции

| Переменная | Описание | По умолчанию |
|-----------|----------|--------------|
| `MIGRATIONS_PATH` | Путь к директории с миграциями | `migrations` |

## Production развертывание

### Рекомендации

1. **Безопасность**:
   - Используйте сильные пароли для БД
   - Включите SSL для подключения к PostgreSQL (`DB_SSLMODE=require`)
   - Настройте firewall для ограничения доступа к БД
   - Используйте секреты для хранения паролей (не храните в коде)

2. **Производительность**:
   - Настройте connection pool в PostgreSQL
   - Используйте reverse proxy (nginx, traefik) для балансировки нагрузки
   - Настройте мониторинг и логирование

3. **Надежность**:
   - Настройте health checks для автоматического перезапуска
   - Используйте orchestration (Kubernetes, Docker Swarm)
   - Настройте резервное копирование БД

### Docker Compose для Production

Создайте `docker-compose.prod.yml`:

```yaml
services:
  postgres:
    image: postgres:12-alpine
    environment:
      POSTGRES_USER: ${DB_USER}
      POSTGRES_PASSWORD: ${DB_PASSWORD}
      POSTGRES_DB: ${DB_NAME}
    volumes:
      - postgres_data:/var/lib/postgresql/data
    networks:
      - app-network
    restart: unless-stopped

  app:
    build:
      context: .
      dockerfile: Dockerfile
    environment:
      DB_HOST: postgres
      DB_USER: ${DB_USER}
      DB_PASSWORD: ${DB_PASSWORD}
      DB_NAME: ${DB_NAME}
      DB_PORT: "5432"
      DB_SSLMODE: require
      GIN_MODE: release
    depends_on:
      postgres:
        condition: service_healthy
    networks:
      - app-network
    restart: unless-stopped
    ports:
      - "8080:8080"

volumes:
  postgres_data:

networks:
  app-network:
    driver: bridge
```

Запуск:
```bash
docker-compose -f docker-compose.prod.yml up -d
```

## Мониторинг

### Health Check

Сервис предоставляет endpoint для проверки состояния:

```bash
GET /health
```

Ответ включает:
- Статус сервиса (`healthy`/`unhealthy`)
- Статус подключения к БД (`ok`/`unavailable`)
- Timestamp

### Логирование

Логи выводятся в формате JSON (по умолчанию) или console. Уровни логирования:
- `debug` - детальная отладочная информация
- `info` - информационные сообщения
- `warn` - предупреждения
- `error` - ошибки

## Troubleshooting

### Проблемы с подключением к БД

1. Проверьте, что PostgreSQL запущен:
   ```bash
   docker ps | grep postgres
   ```

2. Проверьте переменные окружения:
   ```bash
   env | grep DB_
   ```

3. Проверьте логи:
   ```bash
   docker-compose logs app
   ```

### Проблемы с миграциями

Миграции применяются автоматически при старте. Если миграции не применяются:

1. Проверьте путь к миграциям (`MIGRATIONS_PATH`)
2. Проверьте права доступа к БД
3. Проверьте логи приложения

### Проблемы с портами

Если порт 8080 занят, измените `SERVER_PORT`:

```bash
export SERVER_PORT=:8081
```

## Откат изменений

Для отката миграций используйте файлы `.down.sql` из директории `migrations/`. Применяйте их вручную через psql или другой SQL клиент.

## Обновление

1. Остановите сервис:
   ```bash
   docker-compose down
   ```

2. Обновите код:
   ```bash
   git pull
   ```

3. Пересоберите образы:
   ```bash
   docker-compose build
   ```

4. Запустите сервис:
   ```bash
   docker-compose up -d
   ```

Миграции применятся автоматически при старте.


# Postman Collection для PR Reviewer Assignment Service

## Описание

Эта коллекция содержит тесты для всех эндпоинтов API сервиса назначения ревьюеров для Pull Request'ов. Коллекция соответствует спецификации OpenAPI (`api/openapi.yml`) и покрывает все основные сценарии использования API.

## Установка

### Через Postman UI

1. Откройте Postman
2. Нажмите **Import** в левом верхнем углу
3. Импортируйте файл `postman/postman_collection.json`
4. Импортируйте файл `postman/postman_environment.json` (опционально, для удобства работы с переменными)
5. Выберите импортированное окружение в выпадающем списке окружений

### Через командную строку

```bash
# Импорт коллекции
postman collection import postman/postman_collection.json

# Импорт окружения
postman environment import postman/postman_environment.json
```

## Структура коллекции

### 1. Health
- **Health Check** - Проверка работоспособности сервиса
  - Проверяет статус код 200
  - Проверяет наличие поля `status` со значением `ok`
  - Проверяет время ответа < 300ms

### 2. Teams
- **Create Team** - Создание команды с участниками
- **Get Team** - Получение команды
- **Get Team - Not Found** - Проверка обработки несуществующей команды
- **Create Team - Duplicate** - Проверка обработки дубликата команды

### 3. Users
- **Set User Active** - Установка флага активности пользователя
- **Set User Active - Not Found** - Проверка обработки несуществующего пользователя
- **Get User Reviews** - Получение PR'ов пользователя
- **Get User Reviews - Not Found** - Проверка обработки несуществующего пользователя

### 4. Pull Requests
- **Create PR** - Создание PR с автоматическим назначением ревьюеров
- **Create PR - No Reviewers Available** - Создание PR без доступных ревьюеров
- **Create PR - Duplicate** - Проверка обработки дубликата PR
- **Create PR - Author Not Found** - Проверка обработки несуществующего автора
- **Merge PR** - Объединение PR (идемпотентная операция)
- **Merge PR - Idempotent** - Проверка идемпотентности операции merge
- **Merge PR - Not Found** - Проверка обработки несуществующего PR
- **Reassign Reviewer** - Переназначение ревьюера
- **Reassign Reviewer - PR Merged** - Проверка невозможности переназначения после MERGED
- **Reassign Reviewer - Not Assigned** - Проверка обработки случая, когда пользователь не был назначен ревьювером
- **Reassign Reviewer - No Candidate** - Проверка обработки случая, когда нет доступных кандидатов

### 5. E2E Flow
- **Complete Flow** - Полный E2E тест:
  1. Создание команды
  2. Создание PR
  3. Получение ревью пользователя
  4. Merge PR
  5. Попытка переназначения после merge (должна вернуть ошибку)

## Переменные окружения

Коллекция использует следующие переменные:

- `base_url` - Базовый URL API (по умолчанию: `http://localhost:8080`)
- `team_name` - Имя команды (устанавливается автоматически при создании)
- `pr_id` - ID PR (устанавливается автоматически при создании)
- `reviewer_user_id` - ID ревьюера (устанавливается автоматически при создании PR)
- `e2e_*` - Переменные для E2E тестов (автоматически устанавливаются в процессе выполнения)

## Запуск тестов

### Через Postman UI

1. Убедитесь, что сервис запущен:
   ```bash
   docker-compose up
   # или
   make run
   ```

2. Выберите коллекцию в Postman
3. Выберите окружение "PR Reviewer Service - Local"
4. Нажмите **Run** для запуска всех тестов или выберите конкретные тесты
5. Просмотрите результаты выполнения тестов

### Через Newman (CLI)

Newman - это инструмент командной строки для запуска коллекций Postman.

#### Установка Newman

```bash
npm install -g newman
```

#### Запуск коллекции

```bash
# Базовый запуск
newman run postman/postman_collection.json -e postman/postman_environment.json

# Запуск с подробным выводом
newman run postman/postman_collection.json -e postman/postman_environment.json --verbose

# Запуск с HTML отчетом
newman run postman/postman_collection.json -e postman/postman_environment.json -r html --reporter-html-export report.html

# Запуск с JSON отчетом
newman run postman/postman_collection.json -e postman/postman_environment.json -r json --reporter-json-export report.json

# Запуск конкретной папки
newman run postman/postman_collection.json -e postman/postman_environment.json --folder "Teams"
```

#### Интеграция в CI/CD

Пример для GitHub Actions:

```yaml
- name: Run Postman Tests
  run: |
    npm install -g newman
    newman run postman/postman_collection.json \
      -e postman/postman_environment.json \
      -r cli,html \
      --reporter-html-export postman-report.html
```

## Что проверяют тесты

### Статус коды
- Все тесты проверяют корректность HTTP статус-кодов ответов (200, 201, 400, 404, 409, 500)

### Структура ответов
- Проверка наличия обязательных полей в ответах
- Проверка типов данных (массивы, объекты, строки)
- Проверка значений enum полей (например, статус PR: OPEN/MERGED)

### Бизнес-логика
- Проверка корректности назначения ревьюеров (0-2 ревьюера)
- Проверка идемпотентности операции merge
- Проверка невозможности переназначения после MERGED
- Проверка обработки ошибок (NOT_FOUND, PR_EXISTS, TEAM_EXISTS, PR_MERGED, NOT_ASSIGNED, NO_CANDIDATE)

### Производительность
- Проверка времени ответа health check (< 300ms)

### Автоматизация
- Автоматическое сохранение значений переменных для последующих запросов
- Цепочка зависимых тестов в E2E Flow

## Настройка для разных окружений

Для работы с разными окружениями (dev, staging, production) создайте дополнительные файлы окружений:

1. Скопируйте `postman/postman_environment.json`
2. Измените значение `base_url` на соответствующий URL
3. Импортируйте новое окружение в Postman
4. Выберите нужное окружение перед запуском тестов

Пример для staging окружения:

```json
{
  "name": "PR Reviewer Service - Staging",
  "values": [
    {
      "key": "base_url",
      "value": "https://staging.example.com",
      "type": "default",
      "enabled": true
    }
  ]
}
```

## Примечания

- Перед запуском тестов убедитесь, что база данных чистая или используйте уникальные идентификаторы
- Некоторые тесты зависят от выполнения предыдущих (например, E2E Flow)
- Переменные окружения автоматически устанавливаются при успешных запросах
- Для тестов, которые проверяют дубликаты, убедитесь, что соответствующие ресурсы уже существуют
- E2E Flow тест использует уникальные идентификаторы (`e2e_*`), чтобы избежать конфликтов с другими тестами

## Troubleshooting

### Тесты не проходят

1. **Проверьте, что сервис запущен:**
   ```bash
   curl http://localhost:8080/health
   ```

2. **Проверьте переменные окружения:**
   - Убедитесь, что выбрано правильное окружение
   - Проверьте значение `base_url`

3. **Проверьте логи сервиса:**
   ```bash
   docker-compose logs -f server
   ```

4. **Очистите базу данных:**
   ```bash
   docker-compose down -v
   docker-compose up -d
   ```

### Переменные не устанавливаются

- Убедитесь, что тесты выполняются последовательно
- Проверьте, что предыдущие запросы завершились успешно (статус 200/201)
- Проверьте скрипты тестов в разделе "Tests" каждого запроса

## Дополнительные ресурсы

- [Postman Documentation](https://learning.postman.com/docs/)
- [Newman Documentation](https://github.com/postmanlabs/newman)
- [OpenAPI Specification](api/openapi.yml)


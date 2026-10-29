# Mini-CRM: Сервис управления задачами

REST API сервис для управления задачами внутри команд с ролевой моделью доступа, историей изменений, кешированием списков задач в Redis, защитой от состояния гонки через оптимистичные блокировки и SQL-аналитикой.

---

## 🛠 Стек технологий

- **Язык**: Go (1.24+)
- **СУБД**: MySQL 8.0 (InnoDB, ACID транзакции, Foreign Keys, CTE, JSON функции)
- **Кеш**: Redis 7 (с TTL 5 минут и автоматической инвалидацией при модификациях)
- **Контейнеризация**: Docker & Docker Compose
- **Маршрутизация**: Chi v5 (легковесный, производительный, 100% совместим с `net/http`)
- **Документация**: OpenAPI / Swagger 2.0 (встроенный интерактивный Swagger UI)
- **Отказоустойчивость**: Circuit Breaker (`sony/gobreaker`) для работы с Redis
- **Логирование**: Структурированные логи (`log/slog`) в формате JSON/Text
- **Защита API**: Rate Limiter (Token bucket per client IP)

---

## 🏗 Архитектура и возможности

### 1. Ролевая модель доступа (RBAC)
- **Создатель (`owner`)**:
  - Полный доступ в рамках своей команды.
  - Приглашение участников с ролями `admin` и `member` (роль `owner` через приглашение не выдается).
  - Редактирование любых задач команды.
  - Просмотр аналитического SQL-отчета команды.
- **Администратор (`admin`)**:
  - Приглашение участников (`admin` / `member`).
  - Редактирование любых задач команды.
  - Просмотр аналитического отчета команды.
  - Не может изменить роль или удалить создателя.
- **Участник (`member`)**:
  - Создание задач в своей команде.
  - Просмотр задач, истории изменений и комментариев команды.
  - **Редактирование задач**:
    - Если является **создателем**: может менять название, описание, статус и исполнителя (исполнителем может быть назначен только участник той же команды).
    - Если является **исполнителем**: может менять только **статус** и оставлять комментарии. Переназначать задачу или менять описание/название запрещено.
    - Обычный участник не может редактировать чужие задачи, где он не создатель и не исполнитель.
  - Доступ к чужим командам и их задачам/комментариям/аналитике строго изолирован на уровне бизнес-логики.

### 2. Транзакционность и защита от конкурентных обновлений
- **Атомарные транзакции**:
  - Создание команды и привязка создателя с ролью `owner` выполняются в единой транзакции БД.
  - Создание/обновление задачи и фиксация записи в `task_history` выполняются атомарно.
- **Оптимистичные блокировки (Optimistic Locking)**:
  - В таблице `tasks` хранится поле `version`.
  - При обновлении задачи выполняется проверка версии (`UPDATE tasks SET ..., version = version + 1 WHERE id = ? AND version = ?`).
  - Если за время между получением и сохранением задача была изменена другим пользователем, возвращается `409 Conflict`.
- **История изменений (`task_history`)**:
  - Каждое изменение фиксирует `changed_by` (ID пользователя), `changes` (JSON с полями, старыми и новыми значениями) и `created_at`.

### 3. Кеширование в Redis
- Кешируется результат `GET /api/v1/tasks` с TTL **5 минут**.
- Ключ кеша учитывает `team_id`, фильтры (`status`, `assignee_id`) и параметры пагинации (`limit`, `offset`, `cursor`).
- При любом создании или изменении задачи в команде все ключи кеша для этой команды автоматически инвалидируются.
- Запросы к Redis защищены паттерном **Circuit Breaker**: при сбоях или недоступности Redis приложение плавно переключается на работу напрямую с БД без сбоев пользовательских запросов.

### 4. SQL-отчет аналитики команды
Эндпоинт `GET /api/v1/teams/{team_id}/stats`:
- Доступен **только создателю (`owner`) или администратору (`admin`)** команды.
- Реализован **одним осмысленным SQL-запросом с использованием CTE (Common Table Expressions)** без N+1 проблем.
- Включает:
  1. Количество задач по статусам (`tasks_by_status`).
  2. Топ-3 исполнителя по числу закрытых задач за последние 30 дней (`top_3_assignees`).
  3. Среднее время закрытия задач (`avg_closing_time_seconds`).
  4. Общее число комментариев по задачам команды (`total_comments`).

### 5. Дополнительные возможности ("Будет плюсом")
- [x] **Unit-тесты** на бизнес-логику прав доступа и ролевой матрицы.
- [x] **Интеграционный тест** на SQL-отчет с проверкой агрегаций, фильтрации за 30 дней и проверки доступа.
- [x] **Ограничение частоты запросов** (Rate Limiter на IP).
- [x] **Курсорная пагинация** (`cursor` / `next_cursor` по ID).
- [x] **Структурированные логи** (`slog`) с контекстом запросов, UserID, задержкой и кодами ответов.
- [x] **Circuit Breaker** для внешних вызовов/Redis (`sony/gobreaker`).

---

## ⚙️ Конфигурация

Все параметры конфигурируются через переменные окружения (без хардкода):

| Переменная | По умолчанию | Описание |
|---|---|---|
| `SERVER_PORT` | `8080` | Порт HTTP-сервера |
| `APP_ENV` | `development` | Режим работы (`development` / `production`) |
| `DB_HOST` | `localhost` | Хост MySQL |
| `DB_PORT` | `3306` | Порт MySQL |
| `DB_USER` | `crm_user` | Пользователь БД |
| `DB_PASSWORD` | `crm_pass` | Пароль пользователя БД |
| `DB_NAME` | `crm_db` | Имя базы данных |
| `DB_MAX_OPEN_CONNS` | `25` | Макс. открытых соединений к БД |
| `DB_MAX_IDLE_CONNS` | `10` | Макс. неактивных соединений к БД |
| `REDIS_HOST` | `localhost` | Хост Redis |
| `REDIS_PORT` | `6379` | Порт Redis |
| `REDIS_PASSWORD` | `""` | Пароль Redis |
| `REDIS_DB` | `0` | Номер БД Redis |
| `CACHE_TTL_MINUTES` | `5` | Время жизни кеша в минутах |
| `JWT_SECRET` | `secret...` | Секретный ключ для подписи JWT токенов |
| `JWT_TTL_HOURS` | `24` | Время жизни JWT токена в часах |
| `RATE_LIMIT_RPS` | `50` | Ограничение запросов в секунду на IP |
| `RATE_LIMIT_BURST` | `100` | Максимальный всплеск (burst) запросов |

---

## 🚀 Быстрый запуск через Docker Compose

Для запуска приложения, MySQL 8 и Redis одной командой:

```bash
docker compose up --build -d
```

Проверить статус сервисов:
```bash
docker compose ps
```

Посмотреть логи:
```bash
docker compose logs -f app
```

Остановить сервисы:
```bash
docker compose down
```

---

## 📖 Swagger / OpenAPI документация

Интерактивная документация с возможностью тестирования запросов доступна в браузере:
- **Swagger UI**: [http://localhost:8080/swagger/index.html](http://localhost:8080/swagger/index.html)
- **OpenAPI JSON**: [http://localhost:8080/docs/doc.json](http://localhost:8080/docs/doc.json)

---

## 🧪 Тестирование

### Запуск unit-тестов:
```bash
make test
# или
go test -v ./internal/service/...
```

### Запуск интеграционного теста на SQL-отчет:
*(требуется запущенный MySQL, например через `docker compose up -d mysql`)*
```bash
make test-integration
# или
go test -v -count=1 ./tests/integration/...
```

---

## 💻 Примеры HTTP-запросов (cURL)

### 1. Регистрация пользователя
```bash
curl -X POST http://localhost:8080/api/v1/register \
  -H "Content-Type: application/json" \
  -d '{
    "email": "alice@example.com",
    "password": "password123",
    "name": "Alice Smith"
  }'
```

### 2. Аутентификация (вход) и получение JWT
```bash
curl -X POST http://localhost:8080/api/v1/login \
  -H "Content-Type: application/json" \
  -d '{
    "email": "alice@example.com",
    "password": "password123"
  }'
```
*Ответ возвращает токен:*
```json
{
  "token": "eyJhbGciOiJIUzI1NiIsInR5cCI6IkpXVCJ9...",
  "user": { "id": 1, "email": "alice@example.com", "name": "Alice Smith" }
}
```
*Сохраните токен в переменную окружения:*
```bash
export TOKEN="eyJhbGciOiJIUzI1NiIsInR5cCI6IkpXVCJ9..."
```

---

### 3. Создание команды (пользователь становится `owner`)
```bash
curl -X POST http://localhost:8080/api/v1/teams \
  -H "Authorization: Bearer $TOKEN" \
  -H "Content-Type: application/json" \
  -d '{
    "name": "Backend Team"
  }'
```

### 4. Список команд текущего пользователя
```bash
curl -X GET http://localhost:8080/api/v1/teams \
  -H "Authorization: Bearer $TOKEN"
```

### 5. Приглашение участника в команду (`admin` или `member`)
```bash
curl -X POST http://localhost:8080/api/v1/teams/1/invite \
  -H "Authorization: Bearer $TOKEN" \
  -H "Content-Type: application/json" \
  -d '{
    "user_id": 2,
    "role": "member"
  }'
```
*(Также поддерживается приглашение по `"email": "bob@example.com"`)*.

---

### 6. Создание задачи в команде
```bash
curl -X POST http://localhost:8080/api/v1/tasks \
  -H "Authorization: Bearer $TOKEN" \
  -H "Content-Type: application/json" \
  -d '{
    "team_id": 1,
    "title": "Настроить CI/CD",
    "description": "Подготовить пайплайн GitHub Actions",
    "assignee_id": 2
  }'
```

### 7. Получение списка задач с фильтрами и пагинацией
*(Ответ кешируется в Redis на 5 минут)*
```bash
curl -X GET "http://localhost:8080/api/v1/tasks?team_id=1&status=todo&assignee_id=2&limit=20&offset=0" \
  -H "Authorization: Bearer $TOKEN"
```
*Или с курсорной пагинацией:*
```bash
curl -X GET "http://localhost:8080/api/v1/tasks?team_id=1&cursor=5&limit=10" \
  -H "Authorization: Bearer $TOKEN"
```

---

### 8. Обновление задачи (оптимистичная блокировка и права)
```bash
curl -X PUT http://localhost:8080/api/v1/tasks/1 \
  -H "Authorization: Bearer $TOKEN" \
  -H "Content-Type: application/json" \
  -d '{
    "status": "in_progress",
    "version": 1
  }'
```
*Если задача уже была обновлена другим запросом (версия изменилась), сервер вернет `409 Conflict`.*

---

### 9. История изменений задачи
```bash
curl -X GET http://localhost:8080/api/v1/tasks/1/history \
  -H "Authorization: Bearer $TOKEN"
```
*Ответ:*
```json
[
  {
    "id": 1,
    "task_id": 1,
    "changed_by": 1,
    "changed_by_name": "Alice Smith",
    "changes": {
      "status": { "old": null, "new": "todo" },
      "title": { "old": null, "new": "Настроить CI/CD" }
    },
    "created_at": "2026-10-28T19:00:00Z"
  },
  {
    "id": 2,
    "task_id": 1,
    "changed_by": 1,
    "changed_by_name": "Alice Smith",
    "changes": {
      "status": { "old": "todo", "new": "in_progress" }
    },
    "created_at": "2026-10-28T19:05:00Z"
  }
]
```

---

### 10. Добавление комментария к задаче
```bash
curl -X POST http://localhost:8080/api/v1/tasks/1/comments \
  -H "Authorization: Bearer $TOKEN" \
  -H "Content-Type: application/json" \
  -d '{
    "content": "Пайплайн готов, запущено тестирование"
  }'
```

### 11. Список комментариев задачи
```bash
curl -X GET http://localhost:8080/api/v1/tasks/1/comments \
  -H "Authorization: Bearer $TOKEN"
```

---

### 12. SQL-отчет аналитики команды
*(Доступен только `owner` или `admin` команды)*
```bash
curl -X GET http://localhost:8080/api/v1/teams/1/stats \
  -H "Authorization: Bearer $TOKEN"
```
*Пример ответа:*
```json
{
  "team_id": 1,
  "tasks_by_status": [
    { "status": "todo", "count": 3 },
    { "status": "in_progress", "count": 2 },
    { "status": "closed", "count": 12 }
  ],
  "top_3_assignees": [
    { "user_id": 2, "name": "Bob Admin", "closed_count": 7 },
    { "user_id": 3, "name": "Charlie Dev", "closed_count": 4 },
    { "user_id": 4, "name": "Diana QA", "closed_count": 1 }
  ],
  "avg_closing_time_seconds": 18900.5,
  "total_comments": 28
}
```

---

## 🗄 Структура проекта

```
mini-crm/
├── cmd/
│   └── server/
│       └── main.go                  # Точка входа, запуск сервера и graceful shutdown
├── internal/
│   ├── config/                      # Загрузка и валидация конфигурации из переменных окружения
│   ├── domain/                      # Доменные модели, интерфейсы и ошибки
│   ├── handler/
│   │   └── http/                    # HTTP-хендлеры, маршрутизатор Chi, Swagger UI
│   ├── middleware/                  # JWT auth, structured slog logging, rate limiting
│   ├── repository/
│   │   ├── mysql/                   # MySQL репозитории (транзакции, optimistic locking, CTE отчет)
│   │   └── redis/                   # Redis клиент (circuit breaker, кеширование задач, инвалидация)
│   └── service/                     # Сервисный слой с бизнес-логикой и валидацией прав
├── migrations/
│   └── 001_init.sql                 # SQL схема таблиц, индексов и связей
├── docs/
│   └── swagger.json                 # OpenAPI/Swagger спецификация
├── tests/
│   └── integration/
│       └── stats_test.go            # Интеграционный тест SQL-отчета и аналитики
├── docker-compose.yml               # Оркестрация MySQL, Redis и приложения
├── Dockerfile                       # Multi-stage сборка компактного контейнера
├── Makefile                         # Команды для сборки, тестирования и запуска
└── README.md
```

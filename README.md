# Url Shortener

[![Go](https://img.shields.io/badge/Go-%2300ADD8.svg?&logo=go&logoColor=white)](https://go.dev)
[![Postgres](https://img.shields.io/badge/Postgres-%23316192.svg?logo=postgresql&logoColor=white)](https://www.postgresql.org)
[![Docker](https://img.shields.io/badge/Docker-2496ED?logo=docker&logoColor=fff)](https://hub.docker.com/)

Сервис для сокращения ссылок: POST сохраняет URL и отдаёт сокращенную ссылку, GET возвращает оригинал.

## Содержание

- [Quick Start](#quick-start)
- [Демо](#демо)
- [Особенности](#особенности)
- [Конфигурация](#конфигурация)
- [API](#api)
- [Тестирование](#тестирование)

## Quick Start

Требования: Docker Compose. Для запуска без Docker нужны Go 1.26 и PostgreSQL 18.

```bash
# 1. Скопировать .env
cp .env.example .env

# 2. Поднять в compose сервис, БД с миграциями
docker compose up -d

# 3. Проверить
curl -X POST localhost:8080/api/v1/shorten \
  -H "Content-Type: application/json" \
  -d '{"original_url": "https://example.com/some/long/path"}'

# 4. Остановить сервисы
docker compose stop
```


Локально без Docker нужен запущенный PostgreSQL:

```bash
cp .env.example .env
go run ./cmd/url-shortener/main.go          # PostgreSQL
go run ./cmd/url-shortener/main.go -in-mem  # in-memory
```

## Демо

Полный цикл - создать короткий URL и получить оригинал обратно:

```bash
# Создать короткую ссылку
curl -X POST localhost:8080/api/v1/shorten \
  -H "Content-Type: application/json" \
  -d '{"original_url": "https://go.dev/doc/"}'
# {"original_url":"https://go.dev/doc/","short_url":"https://example.com/<some_code>","created_at":"2026-09-20T17:30:00Z"}

# Получить оригинал
curl localhost:8080/api/v1/shorten/<some_code>
# {"original_url":"https://go.dev/doc/","short_url":"https://example.com/<some_code>","created_at":"2026-09-20T17:30:00Z"}
```

## Особенности

| Компонент | Детали |
|---|---|
| Два хранилища | PostgreSQL по умолчанию или in-memory через `-in-mem` |
| Идемпотентность | повторный POST того же URL возвращает существующий код |
| TTL ссылок | истекают через `DB_URLS_TTL`, удаляются фоновой очисткой раз в `DB_CLEAR_FREQUENCY` и при обращении |
| Код из 10 символов | `[a-zA-Z0-9_]`, до 3 попыток при коллизии |
| Валидация | `original_url`: обязателен, 4-1024 символа, валидный URL |
| Middleware | `x-request-id`, structured-логи через zap, recovery от паник |

## Конфигурация

Конфигурация через переменные окружения и `.env`. Пример значений лежит в [`.env.example`](.env.example). Все переменные обязательны, при отсутствии сервис падает при старте с указанием пропущенных переменных.

| Переменная | Назначение |
|---|---|
| `DOMAIN_URL` | префикс поля `short_url` в ответе |
| `HTTP_SERVER_HOST` | адрес прослушивания, в compose всегда `0.0.0.0` |
| `HTTP_SERVER_PORT` | порт сервиса |
| `HTTP_SERVER_TIMEOUT` | таймауты HTTP-сервера |
| `DB_HOST` | хост БД, в compose всегда `postgres` |
| `DB_PORT` | порт БД |
| `DB_USER` | пользователь БД |
| `DB_PASSWORD` | пароль БД |
| `DB_NAME` | имя БД |
| `DB_SSL_MODE` | sslmode подключения к БД |
| `DB_REQUEST_TIMEOUT` | таймаут запросов к БД |
| `DB_URLS_TTL` | время жизни ссылки с момента создания |
| `DB_CLEAR_FREQUENCY` | период фоновой очистки истёкших ссылок |

## API

| Метод | Путь | Тело запроса | Ответ |
|---|---|---|---|
| POST | `/api/v1/shorten` | `{"original_url": "..."}` | `201 {"original_url": "...", "short_url": "...", "created_at": "..."}` |
| GET | `/api/v1/shorten/{code}` | - | `200 {"original_url": "...", "short_url": "...", "created_at": "..."}` |

Истёкший код возвращает `410` с телом `{"error": "provided short url code expired"}`.

## Тестирование

| Уровень | Что покрыто | Команда |
|---|---|---|
| Unit с флагом `-race` | сервис, коллизии, TTL и очистка, in-memory репозиторий, middleware, валидация | `task test-unit` или `go test -race ./...` |

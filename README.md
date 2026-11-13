# SruNal

Соцсеть с лентой, профилями и календарём внутри Telegram Mini App. Личные сообщения остаются в Telegram. В каталоге четыре приложения: **Docgen**, **SruPulse**, **SruNotes**, **SruSplit**. Календарь SruDay открывается отдельной основной вкладкой.

Документация продукта и архитектуры находится в [`docs/`](docs/README.md).

## Стек

| Часть | Решение |
| --- | --- |
| Клиент | React 19, Vite, TypeScript (strict), Tailwind 4 |
| Данные клиента | TanStack Query и `fetch` |
| API | Go, стандартный `net/http`, `log/slog` |
| База | PostgreSQL 18, `pgx/v5` |
| Уведомления | Telegram Bot API из отдельного worker |
| Развёртывание | Docker Compose и Caddy |

## Структура

```text
web/src/
  app/        маршруты, провайдеры, сессия, тема, реестр приложений
  core/       лента, публикации и профили самой SruNal
  apps/       doc, day, pulse, notes, split
  shared/     UI DocGen, HTTP-клиент, Telegram SDK, локализация
  styles/     токены и общие стили
backend/
  cmd/        api, worker, migrate
  internal/
    app/            сборка сервисов и маршрутов
    core/           auth, user, social, notify
    apps/           doc, day, pulse, notes, split
    integration/    подтверждённые действия между приложениями
    platform/       httpapi, postgres, secure, telegram, config, vision
db/migrations/      SQL по этапам
deploy/             Dockerfile, Compose, Caddyfile
docs/               продукт и архитектура
```

Правило: приложение владеет своей папкой на фронтенде и на бэкенде. Прямые импорты между
`apps/*` запрещены, связи собирает `integration`.

## Разработка

```powershell
# фронтенд
cd web
npm install
npm run dev          # http://localhost:5173, /api проксируется на :8080

# бэкенд
cd backend
go build ./...
go test ./...

# база и миграции
docker compose -f deploy/docker-compose.yml up -d db
$env:SRUNAL_DATABASE_URL = "postgres://srunal:...@localhost:5432/srunal?sslmode=disable"
go run ./cmd/migrate -dir ../db/migrations
```

## Секреты

Ключи шифрования, HMAC-индекса, токен бота и секрет webhook задаются только окружением
(`SRUNAL_*`). В репозитории их нет; `.env` и локальные файлы секретов игнорируются Git.

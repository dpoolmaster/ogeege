## 1. Каркас проекта

- [ ] 1.1 Создать `go.mod` с модулем `oge-ege`, структуру каталогов `cmd/server`, `cmd/migrate`, `internal/shared/{config,db,httpserver,logging,ids}`, `migrations`, `docker`, `web/static`; проверить, что `go build ./...` проходит на пустом каркасе.
- [ ] 1.2 Добавить зависимости (`gin`, `pgx v5`, `golang-migrate`, `cleanenv`, `godotenv`, `google/uuid`, `golangci-lint`) и зафиксировать `go.sum`; проверить `go mod tidy` без изменений и успешную сборку.
- [ ] 1.3 Добавить `.gitignore` (бинарники, `.env`, артефакты сборки) и проверить, что `.env` не попадает в индекс.

## 2. Конфигурация

- [ ] 2.1 Реализовать `internal/shared/config`: единый `Config` с группами `App`, `HTTP`, `DB`, `Log`, `SMTP`, `S3`, загрузка через `cleanenv` + `godotenv`, валидация обязательных полей и форматов; проверить unit-тестами полной, неполной и некорректной конфигурации.
- [ ] 2.2 Реализовать fail-fast в `cmd/server/main.go`: при ошибке конфигурации вывести причину и завершиться с ненулевым кодом до открытия порта; проверить запуском без `DATABASE_URL`.
- [ ] 2.3 Создать `.env.example` с переменными этапа 0 (`APP_ENV`, `APP_BASE_URL`, `HTTP_ADDR`, `HTTP_*_TIMEOUT_SECONDS`, `HTTP_MAX_BODY_BYTES`, `LOG_LEVEL`, `SESSION_SECRET`, `DATABASE_URL`, `DB_*_CONNS`, `MIGRATIONS_DIR`, `SMTP_*`, `S3_*`) и пометками обязательности; проверить тестом, что каждая переменная конфига присутствует в примере.

## 3. PostgreSQL и транзакции

- [ ] 3.1 Реализовать `internal/shared/db`: создание `pgxpool` с `DB_MIN_CONNS`/`DB_MAX_CONNS`, `Ping` с таймаутом, закрытие пула; проверить интеграционным тестом подключения к Postgres из Docker.
- [ ] 3.2 Реализовать helper `WithTx(ctx, pool, fn)` с `Begin`/`Rollback`/`Commit` и оборачиванием ошибок через `%w`; проверить интеграционным тестом commit и rollback.
- [ ] 3.3 Определить интерфейс доступа к БД, реализуемый и пулом, и `pgx.Tx`, чтобы репозитории не управляли транзакциями; проверить компиляцией тестового репозитория с обоими типами.

## 4. Миграции

- [ ] 4.1 Создать `migrations/000001_init.up.sql` и `.down.sql`: up создаёт схемы `auth`, `profile`, `content`, `learning`, `ai`, `review`, `progress`, `parent`, `billing`, `notify`, `admin` через `CREATE SCHEMA IF NOT EXISTS`; down удаляет их, не трогая `public`.
- [ ] 4.2 Реализовать `cmd/migrate` с `//go:embed` миграций и командами `up`, `down 1`, `version`; проверить на чистой БД: `up` → `version` → `down 1` → `up`.
- [ ] 4.3 Реализовать проверку состояния миграций для readiness (сравнение последней embedded-версии с `public.schema_migrations`); проверить unit-тестами случаев «применены», «не применены», «таблица отсутствует».

## 5. HTTP-сервер и middleware

- [ ] 5.1 Реализовать `cmd/server/main.go` с Gin, режимом из `APP_ENV`, таймаутами чтения/записи и graceful shutdown по `SIGINT`/`SIGTERM` с `HTTP_SHUTDOWN_TIMEOUT_SECONDS`; проверить запуск, ответ на `/healthz` и корректное завершение по сигналу.
- [ ] 5.2 Реализовать middleware `request_id` (валидация входящего `X-Request-Id`, иначе UUID v7; контекст и заголовок ответа) и проверить тестом через `httptest`.
- [ ] 5.3 Реализовать middleware recovery: логирование паники, ответ `500` в едином конверте ошибки без внутренних деталей; проверить тестом с паникующим обработчиком.
- [ ] 5.4 Реализовать middleware логирования запроса (JSON-запись с методом, путём, статусом, длительностью, `request_id`) и проверить тестом, что секреты и cookie не попадают в лог.
- [ ] 5.5 Реализовать middleware безопасности: `X-Content-Type-Options`, `Referrer-Policy`, `X-Frame-Options`, базовая CSP, HSTS только в production; проверить тестом на наличие заголовков.
- [ ] 5.6 Реализовать ограничение размера тела (`http.MaxBytesReader`, `HTTP_MAX_BODY_BYTES`) с ответом `413`/`bad_request`; проверить тестом превышения лимита.
- [ ] 5.7 Реализовать `/healthz` (без зависимостей) и `/readyz` (БД + миграции) с указанием причины неготовности; проверить тестами с фейковыми `Pinger` и чекером миграций.

## 6. Логирование и идентификаторы

- [ ] 6.1 Реализовать `internal/shared/logging`: `slog` с `JSONHandler`, уровнем из `LOG_LEVEL`, выводом в stdout и helper-функцией редактирования чувствительных полей; проверить тестом формата и редактирования.
- [ ] 6.2 Реализовать `internal/shared/ids` для UUID v7 через `google/uuid`; проверить тестом версию и неубывание при последовательной генерации.

## 7. Локальная инфраструктура

- [ ] 7.1 Создать `docker/Dockerfile` (multi-stage, бинарники `server` и `migrate`, non-root) и проверить сборку образа.
- [ ] 7.2 Создать `docker/docker-compose.yml` с `postgres:16`, `mailhog`, `minio`, `minio-init`, `migrate`, `app`, healthcheck'ами и томами `pgdata`/`miniodata`; проверить `docker compose config` без ошибок.
- [ ] 7.3 Проверить полный локальный сценарий: `make up` → `app` отвечает на `/readyz` → письмо видно в MailHog → после `down`/`up` данные Postgres и MinIO сохраняются.

## 8. Инструменты и CI

- [ ] 8.1 Создать `Makefile` с целями `up`, `down`, `migrate-up`, `migrate-down`, `test`, `lint`, `build`, `templ`; проверить каждую цель локально.
- [ ] 8.2 Добавить конфигурацию `golangci-lint` с запретом `os.Getenv` вне `internal/shared/config` (forbidigo) и проверить, что нарушение валит `make lint`.
- [ ] 8.3 Создать `.github/workflows/ci.yml`: линтер → `templ generate` и проверка отсутствия diff → unit-тесты → сервис Postgres, `migrate up`/`down 1`/`up` → сборка; проверить прогон на ветке с заведомо успешным кодом.
- [ ] 8.4 Реализовать `make templ` как no-op при отсутствии `.templ`-файлов и как `templ generate` при их наличии; проверить оба случая.

## 9. Проверка этапа

- [ ] 9.1 Прогнать `go test ./...` и `make lint`; проверить, что все тесты зелёные и линтер без замечаний.
- [ ] 9.2 Проверить сценарий холодного старта: пустая БД → `make migrate-up` → `make up` → `/healthz` и `/readyz` отвечают `200`, логи в JSON с `request_id`.
- [ ] 9.3 Проверить отказные сценарии: недоступная БД → `/readyz` не готов; отсутствующая переменная → приложение не стартует; неприменённые миграции → `/readyz` не готов.
- [ ] 9.4 Сверить реализацию со спекой `platform-foundation` и `AGENTS.md`: нет бизнес-логики в handler, нет `os.Getenv` вне config, нет TODO/FIXME и неиспользуемых зависимостей.

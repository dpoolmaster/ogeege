# Состояние разработки

Источник правды о том, какие этапы ТЗ реализованы, какие спланированы и что делать следующим.

- ТЗ: `docs/technical-specification.md`, раздел 16 «Поэтапный план реализации».
- Правила кода: `AGENTS.md`.
- Планирование: OpenSpec, `openspec/changes/`.
- Документ обновляется после каждого перехода статуса. Статусы: `не начат` → `спланирован` → `в работе` → `реализован`.
- `спланирован` = артефакты change созданы и валидны, код не написан.
- `реализован` = все задачи change выполнены, тесты и линтер зелёные, change заархивирован.

## Сводка

| Этап | Название | Change | Статус |
|---|---|---|---|
| 0 | Каркас проекта | `stage-0-platform-foundation` | спланирован |
| 1 | Аутентификация, профиль, роли | `stage-1-auth-profile-roles` | не начат |
| 2 | Контент, админка, импорт | `stage-2-content-admin-import` | не начат |
| 3 | Публичный контур | `stage-3-public-site` | не начат |
| 4 | Диагностика, карта пробелов, план | `stage-4-diagnostics-gap-map-plan` | не начат |
| 5 | Практика и автопроверка | `stage-5-practice-autocheck` | не начат |
| 6 | Пробники, таймер, шкалы | `stage-6-trial-exam-scales` | не начат |
| 7 | AI-контур и проверка части 2 | `stage-7-ai-extended-answers` | не начат |
| 8 | Прогресс, прогноз, родитель | `stage-8-progress-forecast-parent` | не начат |
| 9 | Freemium, подписки, YooKassa | `stage-9-freemium-subscriptions-yookassa` | не начат |
| 10 | Уведомления, безопасность, полировка | `stage-10-notifications-hardening` | не начат |

## Следующий шаг

**Этап 1 — Аутентификация, профиль, роли.**

- Спланировать change: промпт `prompts/plan-next-stage.md`.
- Затрагиваемые capability: `auth` (новый), `profile` (новый), `notify` (email-outbox часть).
- Зависимости: этап 0.

## Детали по этапам

### Этап 0. Каркас проекта — S

Change: `stage-0-platform-foundation`.

- Capability: `platform-foundation`.
- Артефакты: `proposal.md`, `specs/platform-foundation/spec.md`, `design.md`, `tasks.md` — готовы, `openspec validate --strict` проходит.
- Код не начат. Задачи: 9 групп (каркас, конфиг, БД/транзакции, миграции, HTTP/middleware, логи/ids, инфраструктура, инструменты/CI, проверка).

### Этап 1. Аутентификация, профиль, роли — M

Change: `stage-1-auth-profile-roles` (не создан).

- Capability: `auth`, `profile`, `notify` (только email-outbox и шаблоны подтверждения/сброса).
- Ключевые требования ТЗ: FR-AUTH-01…10, FR-PROF-01…06, SEC-AUTH-01…06, SEC-RBAC-01…05.
- Зависимости: этап 0.

### Этап 2. Контент, админка, импорт — L

Change: `stage-2-content-admin-import` (не создан).

- Capability: `content`, `admin`.
- Ключевые требования ТЗ: FR-CONT-01…10, FR-ADM-01…12, FR-IMP-01…07, QC-01…06.
- Зависимости: этап 1 (роли, аудит).

### Этап 3. Публичный контур — M

Change: `stage-3-public-site` (не создан).

- Capability: `public-site`.
- Ключевые требования ТЗ: FR-PUB-01…08.
- Зависимости: этап 2. Может идти параллельно этапу 4.

### Этап 4. Диагностика, карта пробелов, план — L

Change: `stage-4-diagnostics-gap-map-plan` (не создан).

- Capability: `learning`.
- Ключевые требования ТЗ: FR-LEARN-01…05, правила освоения, приоритет темы.
- Зависимости: этап 2. Может идти параллельно этапу 3.

### Этап 5. Практика и автопроверка — L

Change: `stage-5-practice-autocheck` (не создан).

- Capability: `grading`, `learning` (practice, review, mistakes).
- Ключевые требования ТЗ: FR-GRADE-01…10, FR-LEARN-06, 08, 09, 11.
- Зависимости: этап 4.

### Этап 6. Пробники, таймер, шкалы — M

Change: `stage-6-trial-exam-scales` (не создан).

- Capability: `learning` (trial, deadline, results), `content` (score scales, variants).
- Ключевые требования ТЗ: FR-LEARN-07, 10, FR-GRADE-20…26, раздел 11.4–11.5.
- Зависимости: этап 5.

### Этап 7. AI-контур и проверка части 2 — XL

Change: `stage-7-ai-extended-answers` (не создан).

- Capability: `ai`, `review`, `learning` (part 2 flow).
- Ключевые требования ТЗ: FR-AI-01…12, FR-REV-01…09, AI-20…25, раздел 9.
- Зависимости: этапы 5 и 6.

### Этап 8. Прогресс, прогноз, родитель — M

Change: `stage-8-progress-forecast-parent` (не создан).

- Capability: `progress`, `parent`.
- Ключевые требования ТЗ: FR-PROG-01…07, FR-PAR-01…07, алгоритм прогноза.
- Зависимости: этапы 5 и 7.

### Этап 9. Freemium, подписки, YooKassa — L

Change: `stage-9-freemium-subscriptions-yookassa` (не создан).

- Capability: `billing`.
- Ключевые требования ТЗ: FR-BILL-01…11, SEC-PAY-01…05, состояния подписки.
- Зависимости: этапы 1 и 2.

### Этап 10. Уведомления, безопасность, полировка — M

Change: `stage-10-notifications-hardening` (не создан).

- Capability: `notify` (in-app, напоминания), `platform-foundation` (метрики, security headers).
- Ключевые требования ТЗ: FR-NOTIF-01…07, SEC-WEB-01…08, NFR-20…24, NFR-40…43.
- Зависимости: все предыдущие.

## Порядок зависимостей

```
0 -> 1 -> 2 -> 3
          |
          v
          4 -> 5 -> 6
                    |
                    v
                    7 -> 8 -> 9 -> 10
```

Схема показывает рекомендуемый последовательный порядок. Минимальные зависимости уточняются в деталях этапов: этапы 3 и 4 могут выполняться параллельно после этапа 2; этап 7 требует этапов 5 и 6; этап 8 — этапов 5 и 7; этап 9 — этапов 1 и 2.

## Журнал изменений

| Дата | Изменение |
|---|---|
| 2026-11-26 | Создан файл. Этап 0 спланирован, этапы 1–10 не начаты. |

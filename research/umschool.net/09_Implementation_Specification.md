# 09. Спецификация реализации

Документ делится на:
- **A. Observed requirements** — что фактически требуется по результатам исследования.
- **B. Recommended implementation** — как это реализовать (рекомендации, не факты об исходной архитектуре).
- **C. Unknown architecture** — внутренние решения UMSCHOOL, которые не удалось установить.

---

# A. Observed requirements

## A.1. Функциональные требования (укрупнённо)

1. **SSO-аутентификация**: телефон + OTP, e-mail + пароль, VK ID, восстановление и смена пароля, подтверждение телефона.
2. **Профиль ученика**: ФИО, телефон, e-mail, VK, Telegram, ник, аватар/инициалы, ID, дата рождения, push-настройки, документы.
3. **Онбординг**: выбор класса (5–11), 3–5 предметов, олимпиадный флаг, завершение.
4. **Каталог**: категории курсов, фильтры (класс/предмет/преподаватель), карточки продуктов, тарифы (Стандарт/Про/Премиум), цены, даты, преподаватели, «от N ₽/мес».
5. **Комплекты**: выбор 2–4/2–8 предметов, преподаватели, срок (1 мес/год/2 года), тариф, скидка.
6. **Корзина**: позиции, подарки/спец. цена, промокод, реферальная скидка, суммы, ссылка на оплату, очистка, счётчик.
7. **Оплата**: checkout с контактами, способы оплаты (СБП, карта, рассрочка, Сплит, T-Pay, Яндекс Пэй, реквизиты, цифровой рубль), скидка за способ, успешная оплата.
8. **Подписки**: активные/неактивные, история, продление, апгрейд, автопродление, способы оплаты, причины отключения.
9. **Обучение**: список курсов/модулей/занятий, плеер (Kinescope/YouTube/GCore), таймкоды, материалы, ДЗ, мини-тесты, оценка занятия.
10. **Домашние работы**: список с фильтрами, старт, таймер, типы заданий (TEST/TEXT/WORD), rich-редактор, ответ/пропуск, завершение/отказ, результаты, работа над ошибками, печать, проверка, доп. проверка.
11. **Тренажёр**: настройки (режим, статусы, избранное, время, лимит, скрыть ответы), сессия карточек, оценки, результат, аналитика, генерация пробников.
12. **Аналитика**: предсказание балла, темы, номера КИМ, пробные варианты, поиск по темам.
13. **Геймификация**: уровни/опыт (61 уровень), умкоины/медали/ключи, достижения (8), ежедневная серия, ежедневные/недельные/месячные задания, купоны, мерч.
14. **Реферальная программа**: ссылка/QR, условия, вознаграждения, баланс, история, вывод, обратная связь.
15. **Чат**: топики по предметам, сообщения, поиск, непрочитанные, передача куратору, рабочие часы.
16. **Уведомления**: push, e-mail, SMS, in-app модалки, системные модалки, промо.
17. **Документы**: публичные (оферта, политика, лицензия), личные, по продуктам, чеки.
18. **Публичный сайт**: лендинги, каталог, преподаватели, отзывы, FAQ, контакты, способы оплаты, промокоды, банк заданий (76k+), пробники, тренажёры, эмулятор, лид-формы.
19. **Админка/роли**: управление курсами, уроками, ДЗ, подписками, платежами, проверками (не проверено).

## A.2. Нефункциональные требования (наблюдаемые/выведенные)

- **Масштаб**: 700 000+ учеников (публичные заявления), 76 158 публичных заданий, 42 835 карточек в тренажёре.
- **Производительность**: SPA, CDN, кэширование, пагинация, lazy-loading чанков.
- **Доступность**: error-страницы 400/403/404/500/502, технические работы.
- **Безопасность**: SSO, JWT, серверная авторизация, CSRF-токены (Django-подобные cookie `csrftoken`), SmartCaptcha.
- **Мультиплатформенность**: RU/ONE/KZ/HEAD, мобильный web, приложения.
- **Локализация**: русский язык; мультиязычность не требуется (по AGENTS.md проекта).
- **Наблюдаемость**: Sentry, Яндекс Метрика, внутренние логи.

---

# B. Recommended implementation

Рекомендация учитывает AGENTS.md проекта: **Go, модульный монолит, PostgreSQL, pgx v5, golang-migrate, без ORM, Templ/HTMX/Pico, Vue SPA для админки**. Для тяжёлых интерактивных экранов (тренажёр, rich-редактор ДЗ, чат, календарь) допускаются Vue-острова.

## B.1. Архитектура

```
cmd/server/main.go
internal/
  auth/            OTP, e-mail, VK ID, JWT, сессии
  profile/         профиль, контакты, настройки
  catalog/         курсы, продукты, тарифы, фильтры
  cart/            корзина, промокоды, комплекты
  orders/          заказы, платежи, чеки
  subscriptions/   подписки, автопродление
  learning/        курсы, модули, занятия, материалы
  homework/        ДЗ, задания, ответы, проверка
  trainer/         карточки, сессии, аналитика
  progress/        знания, темы, КИМ, пробники
  gamification/    уровни, валюты, достижения, серии
  referral/        ссылки, бонусы, выплаты
  messaging/       топики, сообщения, WebSocket
  notifications/   push/email/sms/in-app
  documents/       документы, чеки
  publicsite/      лендинги, банк заданий, отзывы
  admin/           админка (Vue SPA + API)
  shared/          деньги, время, ID, ошибки, пагинация
pkg/
  db/              pgx pool, tx helper
  httpserver/      middleware, CSRF, RBAC
  queue/           jobs (Postgres/Redis)
  storage/         S3
  mailer/          e-mail
  sms/             SMS
  push/            web-push
```

Модуль владеет своей схемой PostgreSQL (`auth`, `catalog`, …), без FK между схемами. Межмодульное взаимодействие — через интерфейсы, объявленные потребителем, сборка — в `main.go`.

## B.2. Frontend

- **Публичный сайт + LMS**: Templ + HTMX + Pico.css для навигации, списков, форм, профиля, каталога, документов.
- **Rich-острова на Vue 3** (или Web Components):
  - плеер и таймкоды;
  - rich-редактор ДЗ (TipTap v3, MathType/ChemType-аналог);
  - тренажёр карточек ( flip-анимации, клавиатура );
  - чат (WebSocket);
  - календарь;
  - checkout (если нужен сложный state).
- **Админка**: Vue 3 composition + TipTap v3 + highlight.js (по AGENTS.md).
- HTMX-паттерны: `hx-get` для пагинации/фильтров, `hx-post` для форм, `hx-swap` таргетно, прогрессивное улучшение.
- Состояния: пустые/загрузка/ошибка/disabled — серверные партиалы.

## B.3. Backend API (группы)

Публичные:
- `GET /api/public/courses`, `/courses/{code}`, `/products/{id}`, `/teachers`, `/reviews`, `/faq`, `/documents`.
- `GET /api/public/task-bank/tasks`, `/task-bank/filters`, `/trial-variants`.
- `POST /api/public/leads`.

Аутентификация:
- `POST /api/auth/otp/request`, `/otp/verify`, `/login`, `/refresh`, `/logout`, `/password/reset`, `/password/change`, `/vk/callback`.

Ученик:
- `GET/PATCH /api/me`, `/me/settings`, `/me/onboarding`.
- `GET /api/catalog`, `/catalog/{code}`, `/products/{id}`, `/products/{id}/rate-plans`.
- `GET/POST/PATCH/DELETE /api/cart`, `/cart/coupon`, `/cart/clear`, `/cart/items`.
- `GET/POST /api/orders`, `/orders/{id}/pay`, `/orders/{id}/receipt`.
- `GET /api/subscriptions`, `/subscriptions/{id}`, `POST /renew`, `/upgrade`, `/auto-renewal`.
- `GET /api/lessons`, `/lessons/{id}`, `/lessons/{id}/materials`, `/lessons/{id}/homework`.
- `GET/POST /api/homework/submissions`, `/start`, `/answer`, `/finish`, `/refuse`, `/results`, `/work-on-mistakes`, `/print`.
- `GET /api/trainer/settings`, `/cards`, `POST /trainer/start`, `/answer`, `/finish`, `GET /trainer/result`.
- `GET /api/progress`, `/progress/topics`, `/progress/kim`, `/progress/trials`, `POST /progress/trials/generate`.
- `GET /api/gamification/balance`, `/levels`, `/achievements`, `/strike`, `/tasks`, `POST /coupons/redeem`.
- `GET/POST /api/referral/link`, `/balance`, `/history`, `/withdraw`.
- `GET/POST /api/chat/topics`, `/messages`, `WS /ws/chat`.
- `GET/PATCH /api/notifications/settings`, `POST /push/subscribe`.

Админ:
- CRUD курсов, модулей, занятий, ДЗ, заданий, тарифов, промокодов, пользователей, проверок, выплат.

## B.4. База данных (рекомендуемая)

Схемы и ключевые таблицы:
- `auth`: `users`, `credentials`, `otp_codes`, `sessions`, `refresh_tokens`, `oauth_identities`.
- `profile`: `profiles`, `contact_preferences`, `consents`.
- `catalog`: `class_years`, `class_types`, `course_types`, `product_types`, `products`, `rate_plans`, `features`, `product_features`.
- `cart`: `carts`, `cart_items`, `cart_gifts`, `coupons`, `coupon_redemptions`.
- `orders`: `orders`, `order_items`, `payments`, `payment_methods`, `receipts`, `refunds`.
- `subscriptions`: `subscriptions`, `subscription_periods`, `auto_renewal_settings`, `renewal_attempts`.
- `learning`: `modules`, `lessons`, `lesson_materials`, `lesson_progress`, `mini_tests`.
- `homework`: `homeworks`, `tasks`, `task_variants`, `submissions`, `answers`, `submission_results`, `curator_reviews`.
- `trainer`: `cards`, `card_statuses`, `training_sessions`, `training_answers`, `training_results`.
- `progress`: `knowledge_categories`, `topics`, `subtopics`, `kim_numbers`, `trial_variants`, `trial_tasks`.
- `gamification`: `levels`, `achievements`, `achievement_levels`, `user_achievements`, `balance_transactions`, `daily_strikes`, `individual_tasks`.
- `referral`: `referral_links`, `referral_events`, `referral_balances`, `withdrawals`.
- `messaging`: `chat_topics`, `chat_messages`, `chat_participants`, `chat_reads`.
- `notifications`: `notifications`, `notification_preferences`, `push_subscriptions`, `email_log`, `sms_log`.
- `documents`: `documents`, `user_documents`, `receipts`.
- `public`: `task_bank`, `task_bank_filters`, `reviews`, `faq`, `leads`.

Деньги — `numeric(12,2)`, валюта RUB. ID — UUID v7 (по AGENTS.md), кроме внешних числовых ID, если нужна совместимость.

## B.5. Фоновые задачи

- Напоминания о занятиях/дедлайнах (за N минут/часов).
- Автопродление подписок и retry при ошибке.
- Сверка платежей с провайдерами (reconciliation).
- Начисление реферальных бонусов после оплаты.
- Пересчёт достижений/уровней/серий (ежедневно).
- Генерация отчётов/чеков.
- Очистка OTP, сессий, временных файлов.
- Индексация поиска/кэша.
- Рассылки (email/SMS/push) с идемпотентностью.

## B.6. Аутентификация/авторизация

- OIDC-подобный провайдер или собственный SSO.
- Access JWT (15 мин) + refresh (30–120 дней) с rotation.
- HttpOnly Secure SameSite cookies для веб-сессий; CSRF-токен для форм.
- RBAC + ownership checks на каждом запросе.
- OTP: TTL 5–10 минут, лимит попыток, rate limit по телефону/IP.
- VK ID OAuth с state/nonce.
- Пароли — Argon2id/bcrypt, минимум 8–12 символов (в исходнике 6, рекомендуется больше).
- Аудит действий администраторов.

## B.7. Безопасность

- Валидация всех входных данных (структурная + бизнес).
- Экранирование HTML; sanitize rich-контента (TipTap → HTML) через allowlist.
- CSP, HSTS, X-Frame-Options, Referrer-Policy.
- Ограничение загрузок (тип, размер, антивирус).
- Идемпотентность платежей и вебхуков (idempotency key).
- Секреты — только в env/секретах, не в коде.
- PII: шифрование чувствительных полей, маскирование в логах, доступ по принципу минимальных привилегий.
- Защита от enumeration (телефон/почта) — нейтральные ответы.
- Rate limiting на login/OTP/промокоды/поиск.

## B.8. Поиск

- PostgreSQL `tsvector` + `pg_trgm` для публичного каталога, отзывов, FAQ, банка заданий.
- Фильтры — индексируемые колонки/связи.
- Кэш популярных запросов (Redis) при росте.
- При больших объёмах — отдельный поисковый индекс (OpenSearch) как Phase 3.

## B.9. Медиа и файлы

- S3-совместимое хранилище, CDN, подписанные URL.
- Видео — внешний провайдер (Kinescope/YouTube/GCore) или собственный HLS.
- Изображения — ресайз/WebP при загрузке.

## B.10. Наблюдаемость

- `log/slog` (JSON), request_id, user_id (без PII).
- OpenTelemetry → трассировка, метрики Prometheus.
- Sentry для ошибок фронта/бэка.
- Алерты: 5xx, рост ошибок платежей, задержки очередей.
- Аудит-лог админских действий.

## B.11. Масштабирование

- Stateless приложение за балансировщиком, горизонтальное масштабирование.
- Read replicas PostgreSQL для каталога/аналитики.
- Redis для сессий/кэша/очередей.
- Очереди для тяжёлых задач (генерация вариантов, проверка, рассылки).
- CDN для статики и видео.
- Партиционирование крупных таблиц (answers, balance_transactions, notifications) по времени.

---

# C. Unknown architecture

Следующие решения UMSCHOOL неизвестны и не должны выдаваться за факты:

- Язык/фреймворк backend (признаки Django: `csrftoken`, `sessionid`, ответы `{"detail": ...}`; но это только предположение).
- СУБД и схема хранения (вероятно PostgreSQL, но не подтверждено).
- Очереди/брокеры (Kafka/RabbitMQ/Redis) и внутренние сервисы.
- Реализация проверки ДЗ «Умкой» (AI-модель, критерии, хранение).
- Алгоритм интервального повторения карточек.
- Формулы предсказания балла и уровня знаний.
- Механика скидок и ценообразования (A/B, региональные цены).
- Архитектура платежей и антифрод.
- Внутренняя модель ролей и админки.
- Точные SLA, ёмкость, региональная инфраструктура.
- Политика хранения и удаления данных.
- Наличие родительского кабинета и его права.

---

## D. Рекомендуемый порядок реализации

1. Auth + профиль + онбординг.
2. Каталог + продукты/тарифы + корзина + checkout (без реального эквайринга на старте — sandbox).
3. Обучение: курсы/модули/занятия/плеер.
4. Домашние работы + результаты + работа над ошибками.
5. Тренажёр карточек.
6. Подписки/автопродление.
7. Геймификация + реферал.
8. Чат + уведомления.
9. Публичный сайт/банк заданий/пробники.
10. Админка + аналитика + масштабирование.
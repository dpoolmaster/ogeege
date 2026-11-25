# 08. Уведомления и интеграции

## 1. Уведомления

### 1.1 Типы (OBSERVED из кода)

| Тип | Канал | Получатель | Триггер | Связанная сущность | Текст/шаблон |
|---|---|---|---|---|---|
| `chat_message` | in-app/push | ученик | новое сообщение наставника | Chat/Message | «Сообщение от наставника» |
| `homework_deadline_today` | in-app/push | ученик | дедлайн ДЗ сегодня | Homework | «Домашняя работа» |
| `homework_deadline_1d` | in-app/push | ученик | дедлайн через 1 день | Homework | «Домашняя работа» |
| `homework_checked` | in-app/push | ученик | ДЗ проверено | Homework/Review | «Домашняя работа» |
| `probnik_deadline_1d` | in-app/push | ученик | пробник через 1 день | Sample | «Пробник» |
| `probnik_checked` | in-app/push | ученик | пробник проверен | Sample/Review | «Пробник» |
| `webinar_starting_soon` | in-app/push | ученик | скоро эфир | Lesson | «Вебинар» |
| `subscription_ending` | in-app/push/email? | ученик/родитель | подписка заканчивается | Subscription | «Подписка» |

### 1.2 Каналы

| Канал | Подтверждение | Комментарий |
|---|---|---|
| In-app (лента) | OBSERVED | `/notifications`, «Пока нет уведомлений» |
| WebSocket (real-time) | OBSERVED | Centrifugo, `wss://notify.devcotka.space/connection/websocket` |
| SMS | CONFIRMED | «Мы отправили смс-уведомление с кодом на номер …» |
| Email | CONFIRMED | «Мы отправили письмо с кодом на почту …» |
| Push | INFERRED | типы уведомлений, но провайдер не обнаружен |
| Telegram | INFERRED | внешние каналы/боты |
| MAX | OBSERVED | ссылка `max.ru` в контактах |

### 1.3 Состояния уведомления
- `unread` / `read` (archive), счётчик непрочитанных, фильтр «Все/Непрочитанные» (для чата).
- Хранение: `Notification` с `read_at`.

### 1.4 Настройки уведомлений
- Отдельный экран настроек не обнаружен; вероятно, управление в профиле (INFERRED). Статус: UNKNOWN.

---

## 2. Интеграции

### 2.1 Платежи

| Интеграция | Назначение | Где | Данные | Статус |
|---|---|---|---|---|
| **CloudPayments** | оплата картой, рассрочка | `/sotkastore` (`widget.cloudpayments.ru`) | сумма, карта (токенизация), статус | OBSERVED |
| Банки-партнёры (рассрочка) | оплата частями | checkout/FAQ | заявка, одобрение | OBSERVED (текст) |
| Внутренние `store/*` API | доступы, продления | ЛК/магазин | product, option, user | INFERRED |

> Реальные платежи не проводились.

### 2.2 Real-time / чат

| Интеграция | Назначение | Данные | Статус |
|---|---|---|---|
| **Centrifugo** | WebSocket-токены, real-time | `GET /api/v3/centrifugo/token` | OBSERVED |
| **Pusher protocol** | клиентская библиотека | `wss://notify.devcotka.space/connection/websocket` | OBSERVED |
| HTTP-чаты | история/отправка | `/chat/*` | INFERRED |

### 2.3 Файлы и медиа

| Интеграция | Назначение | Данные | Статус |
|---|---|---|---|
| **Yandex Cloud Object Storage** | изображения заданий, файлы | `sotka-storage.storage.yandexcloud.net` (presigned URL, X-Amz-*) | OBSERVED |
| CDN платформы | статика | `platform.sotkaonline.ru/storage/...` | OBSERVED |
| Видеохостинг | видеоуроки | URL не обнаружен | UNKNOWN |

### 2.4 Аналитика и маркетинг

| Сервис | Назначение | Признак | Статус |
|---|---|---|---|
| Яндекс.Метрика | веб-аналитика | счётчики 87244457, 84474187, 69070579 | OBSERVED |
| Google Analytics 4 | веб-аналитика | `G-SF8FNRX6Y2` | OBSERVED |
| VK Pixel/Retargeting | реклама | `vk.com/rtrg`, `VK-RTRG-...` | OBSERVED |
| Mail.ru top-fwz1 | реклама | счётчики | OBSERVED |
| AdRiver | реклама | `ad.adriver.ru` | OBSERVED |
| Hybrid.ai | реклама | `dss.hybrid.ai` | OBSERVED |
| Flocktory | маркетинг/ретаргетинг | `api.flocktory.com` | OBSERVED |
| sjsmartcontent | рекомендации | `sjsmartcontent.ru` | OBSERVED |
| targetads.io | аналитика | `eye.targetads.io` | OBSERVED |

### 2.5 Коммуникации и соцсети

| Интеграция | Назначение | Признак | Статус |
|---|---|---|---|
| VK ID / VK | вход/соцсети, VK-ссылка в профиле | `id.vk.ru` (Mail), `vk.com` | OBSERVED |
| Telegram | каналы, боты, поддержка | `t.me/sotka100tg`, `t.me/sotkacormit` | OBSERVED |
| MAX | мессенджер | `max.ru` | OBSERVED |
| YouTube | преподаватели/контент | ссылки в профилях | OBSERVED |
| Почта России? | — | не обнаружено | — |

### 2.6 Профориентация

| Интеграция | Назначение | Признак | Статус |
|---|---|---|---|
| `хочустать.рф` | профориентация | внешняя ссылка | OBSERVED |
| `GET /hochuStat/getUniversity` | вузы | API | INFERRED |

### 2.7 Карты

| Интеграция | Назначение | Признак | Статус |
|---|---|---|---|
| Яндекс.Карты | адрес на `/about/contacts` | «Построить маршрут», «© Яндекс» | OBSERVED |

### 2.8 Юридические/платежные документы

- Оферты по срокам (15/61/90/123/150 дней), ВУЗ, СПО, ССУЗ, возврат.
- Pay-agreement страницы с загрузкой документов и кодом подтверждения.
- Статус: OBSERVED (роуты/страницы), подписание NOT_TESTED.

---

## 3. События и хуки (для реализации)

| Событие | Источник | Реакция |
|---|---|---|
| `payment.succeeded` | CloudPayments webhook | создать StoreAccess, аккаунт, уведомление |
| `payment.failed` | CloudPayments | статус ошибки, retry |
| `homework.submitted` | ЛК | автопроверка / очередь наставника |
| `homework.checked` | наставник | уведомление, статистика |
| `duel.finished` | ЛК | результат, уведомление |
| `subscription.expiring` | cron | уведомление, продление |
| `referral.paid` | оплата друга | начисление, доступность через 30 дней |
| `chat.message` | чат | real-time push |

## 4. Ограничения исследования

- Push-провайдер не определён.
- SMS-провайдер не определён.
- Email-провайдер не определён.
- Видеохостинг не определён.
- Реальные вебхуки и подписи не проверялись.
# 08. Уведомления и интеграции

---

## 1. Каналы уведомлений

| Канал | Подтверждение | Комментарий |
|---|---|---|
| Web push | CONFIRMED | `/webpush/service-worker.js`, `/api/v2/core/webpush/subscribe|unsubscribe/` |
| Mindbox webpush | CONFIRMED (код) | `/api/v2/core/mindbox/webpush/`, `/mindboxwebpush/service-worker.js` |
| E-mail | CONFIRMED (сценарии) | Подтверждение почты, восстановление пароля; провайдер не раскрыт |
| SMS | CONFIRMED (сценарий) | Код входа по телефону; провайдер не раскрыт |
| In-app модалки | CONFIRMED | Push-модалка, системные модалки, промо |
| Чат/куратор | CONFIRMED | `chat-api`, WebSocket/Centrifugo |
| Мессенджеры | OBSERVED | Telegram-канал преподавателя, VK-бот/чат поддержки |
| Push через notifications WS | OBSERVED | `wss://notifications.umschool.net/messenger/` |

**Внутреннего центра уведомлений (списка «колокольчик») в LMS не обнаружено.** Уведомления — это push, модалки, чат и e-mail/SMS.

---

## 2. Таблица уведомлений

| ID | Триггер | Получатель | Канал | Содержание | Условия | Связанная сущность | Действия |
|---|---|---|---|---|---|---|---|
| N-01 | Первый вход/периодически | Student | In-app модалка | «Хотите получать уведомления? Остаёмся на связи… Включить уведомления / Не сейчас» | Пока не разрешено/не отклонено | User | Разрешить/отклонить |
| N-02 | Включение push | Student | Web push | Регистрация подписки | Разрешение браузера | PushSubscription | — |
| N-03 | Отключение push | Student | Web push | `unsubscribe` | — | PushSubscription | — |
| N-04 | Приближение занятия | Student | Push/E-mail (предположительно) | Напоминание о занятии | Есть расписание | Lesson | Перейти к занятию |
| N-05 | Новое ДЗ / дедлайн | Student | Push/E-mail | Напоминание о ДЗ | Есть ДЗ | Homework | Открыть ДЗ |
| N-06 | Проверка ДЗ | Student | Push/чат | «ДЗ проверено», комментарии | Проверка завершена | HomeworkSubmission | Открыть результат |
| N-07 | Ответ куратора | Student | Chat/WS | Новое сообщение | Открыт диалог | ChatMessage | Ответить |
| N-08 | Непрочитанные сообщения | Student | Бейдж | Счётчик | Есть непрочитанные | ChatTopic | Открыть чат |
| N-09 | Подтверждение телефона | Student | In-app модалка/SMS | «Необходимо подтвердить номер…» | `is_phone_confirmed=false` | User | Подтвердить |
| N-10 | Подтверждение e-mail | Student | E-mail | Ссылка/код | `is_email_confirmed=false` | User | Подтвердить |
| N-11 | Согласие на ПД | Student | In-app | Политика/согласие | `is_personal_data_agreement=false` | User | Принять |
| N-12 | Рекламные рассылки | Student | E-mail/SMS | Маркетинг | `accepted_advertising_mail` | User | Отписаться |
| N-13 | Новая ачивка/уровень | Student | In-app/push | «Новая награда», «Новый уровень» | Достигнут порог | Achievement/Level | Посмотреть |
| N-14 | Ежедневная серия | Student | In-app | Прогресс серии, награда | `strike_days` | DailyStrike | — |
| N-15 | Индивидуальные задания | Student | In-app | День/неделя/месяц, дедлайн | Есть задания | IndividualTask | Выполнить |
| N-16 | Промо-акция | Student | Popmechanic | «ВЕБЫ ИДУТ БЕЗ ТЕБЯ», таймер, «Записаться» | Кампания | Campaign | Перейти в Telegram |
| N-17 | Оплата | Student | E-mail/push | «Оплата прошла успешно», доступ открыт | Успешная транзакция | Payment | Перейти в курс |
| N-18 | Ошибка оплаты | Student | In-app/e-mail | «Произошла ошибка при оплате» | Ошибка | Payment | Повторить/другой способ |
| N-19 | Истечение подписки | Student | E-mail/push | «Если не продлить, потеряешь преимущества» | Окончание периода | Subscription | Продлить |
| N-20 | Автопродление | Student | E-mail/push | «Автопродление включено/отключено» | Изменение настройки | Subscription | — |
| N-21 | Возврат | Student | E-mail | Комиссия платёжной системы, остаток вернётся | Возврат | Payment | — |
| N-22 | Технические работы | Все | In-app/страница | «На платформе идут технические работы» | 502 | — | — |
| N-23 | Ошибка в задании | Student | In-app | «Мы получили сообщение…» | Отчёт об ошибке | Task | — |
| N-24 | Передача диалога | Student | Chat | «Диалог передан куратору» | Передача | ChatTopic | — |
| N-25 | Дедлайн ДЗ близко | Student | Push | «Дедлайн прошёл/скоро» | `deadline_is_burning` | Homework | — |

Точные шаблоны и timing не раскрыты; таблица составлена по строкам локали, API-флагам и наблюдаемым модалкам.

---

## 3. Системные модалки (API `system-modals`)

Ответ на 25.11.2026:

```json
{
  "is_social_account_confirmed": true,
  "is_phone_confirmed": false,
  "personal_data_status": null,
  "is_personal_data_agreement": false
}
```

Следствия:
- модалка подтверждения телефона;
- возможная модалка согласия на обработку ПД;
- флаг соцаккаунта подтверждён.

---

## 4. Интеграции

### 4.1. Платежи

| Интеграция | Назначение | Где | Данные | Статус |
|---|---|---|---|---|
| Alfa Bank (`ALFA`, `ALFA_SBP`) | Основной платёжный шлюз, СБП | `pay.umschool.net`, checkout | сумма, заказ, плательщик | CONFIRMED (publicId `umschool-api`) |
| CloudPayments (`CP`) | Оплата картой | checkout | publicId `pk_b750...` | CONFIRMED (конфиг) |
| Yandex Pay (`YA`) | Оплата/QR | checkout | merchantId `a38fc4d4-...` | CONFIRMED |
| Sber (`SBER`) | Рассрочка | checkout | — | CONFIRMED (enum/страница) |
| Tinkoff/T-Bank (`TNKF`) | Рассрочка | checkout | — | CONFIRMED (enum/страница) |
| JetPay (`JP`) | Оплата | checkout | — | CONFIRMED (enum) |
| Kaspi (`KSPI`) | Оплата в Казахстане | checkout | — | CONFIRMED (enum) |
| Pgw (`PGW`) | Платёжный шлюз | checkout | publicId `1` | CONFIRMED (конфиг) |
| СБП | QR-оплата | checkout | — | CONFIRMED (UI) |
| Цифровой рубль | Оплата | checkout | — | CONFIRMED (UI) |
| По реквизитам | Физ/юрлица, чек | checkout/страница | реквизиты, чек | CONFIRMED |

### 4.2. Видео

| Интеграция | Назначение | Где | Данные | Статус |
|---|---|---|---|---|
| Kinescope | Видеоплеер занятий | `/lesson/:id` | `kinescope.io/...` | CONFIRMED |
| YouTube | Видео | уроки | youtube.com/youtu.be | CONFIRMED (код) |
| GCore / CDN UMSchool | Видео/статика | уроки, CDN | cdn*.umschool.* | CONFIRMED (код) |

### 4.3. Чат и real-time

| Интеграция | Назначение | Где | Данные | Статус |
|---|---|---|---|---|
| `chat-api.umschool.net` | REST чата | `/chat` | топики, сообщения | CONFIRMED |
| Centrifugo `wss://chat-centrifugo.umschool.net/connection/websocket` | Real-time чат | `/chat` | публикации | CONFIRMED (код) |
| `wss://chat.umschool.net/messenger/` | WebSocket сообщений | чат | — | CONFIRMED (код) |
| `wss://notifications.umschool.net/messenger/` | WebSocket уведомлений | приложение | — | CONFIRMED (код) |
| VK (`vk.me/umschoolhelp`) | Чат поддержки | публичный сайт | — | CONFIRMED (ссылка) |

### 4.4. Аутентификация

| Интеграция | Назначение | Где | Данные | Статус |
|---|---|---|---|---|
| `id.umschool.net` | SSO | вход | OAuth client_id, токены | CONFIRMED |
| VK ID | Соцвход | login/profile | OAuth | CONFIRMED |
| Дневник.ру | Callback-интеграция | `/dnevnik-callback` | параметры | OBSERVED |
| DEC | Callback | `/dec-callback` | — | OBSERVED |
| SmartCaptcha (Яндекс) | Защита форм | login | токен | CONFIRMED (скрипт) |

### 4.5. Аналитика и маркетинг

| Интеграция | Назначение | Где | Данные | Статус |
|---|---|---|---|---|
| Яндекс Метрика | Веб-аналитика | все страницы | счётчики 61783690, 26812653, 84474187, 69070579, 87244457 (часть — публичный сайт) | CONFIRMED |
| Mindbox | CDP, webpush, сегментация | все страницы | deviceUUID, customer | CONFIRMED |
| Flocktory | Маркетинг/ретаргетинг | все страницы | site_id 3749 | CONFIRMED |
| CoMagic | Коллтрекинг/аналитика | публичный сайт | `_comagic_id` | CONFIRMED (cookie) |
| TargetAds | Реклама/пиксели | все страницы | `_tads_*` | CONFIRMED |
| Top.Mail.Ru | Аналитика | все страницы | counter 3209527 | CONFIRMED |
| Advcake | Партнёрская аналитика | все страницы | `advcake_*` | CONFIRMED |
| GetIQ / Hybrid / RAP / Reffection | Пиксели/реклама | публичный сайт | — | CONFIRMED (скрипты) |
| Popmechanic | Попапы/кампании | все страницы | таймеры, сегменты | CONFIRMED |
| Sentry | Ошибки | SPA | DSN | CONFIRMED (конфиг) |

### 4.6. Инфраструктура и хранение

| Интеграция | Назначение | Где | Данные | Статус |
|---|---|---|---|---|
| S3 `s3.ru-msk.k2.cloud` | Файлы, документы, PDF | документы, медиа | — | CONFIRMED |
| CDN `egumcdn/egndcdn/cdn.umschool.net` | Статика/изображения | все страницы | — | CONFIRMED |
| Kubernetes/nginx | Хостинг | `umschool.net` | заголовки `x-backend: kubernetes` | CONFIRMED (заголовки) |
| DDoS-Guard | Защита `umschool.one` | лендинг | cookies `__ddg*` | CONFIRMED |
| Cloudflare/иное | Защита `umschool.net` | — | `__ddg1_` | CONFIRMED (cookie) |

### 4.7. Прочее

| Интеграция | Назначение | Статус |
|---|---|---|
| `store.umschool.net` | Магазин мерча | CONFIRMED (ссылка) |
| `support.umschool.net` | База знаний | OBSERVED |
| Telegram-боты (`t.me/predbanmanager_bot`, каналы преподавателей) | Промо/связь | CONFIRMED |
| Мобильное приложение (Rustore, iOS PWA) | Мобильный доступ | OBSERVED |

---

## 5. Что неизвестно

- Точные шаблоны писем/SMS и их триггеры.
- Провайдеры SMS/e-mail.
- Расписание отправки напоминаний (за сколько минут/часов).
- Наличие push-кампаний по сегментам Mindbox.
- Полный список вебхуков и внутренних событий.
- Настройки приватности и согласий на уровне БД.
- SLA доставки и retry-политики.

---

## 6. Рекомендации по реализации

- Ввести единый `notification` сервис: событие → шаблон → канал (push/email/sms/in-app).
- Хранить `notification_preferences` пользователя (каналы, типы, quiet hours).
- Для push — VAPID/web-push + service worker; для e-mail — транзакционный провайдер; для SMS — российский провайдер с fallback.
- Для in-app — таблица `notifications` + счётчик непрочитанных + WebSocket.
- Для чата — отдельный сервис (WebSocket/Centrifugo), не смешивать с уведомлениями.
- Для платежей — идемпотентные вебхуки, статусы `pending/succeeded/failed/refunded`.
- Логировать согласия (ПД, реклама, оферта) с версией документа и датой.
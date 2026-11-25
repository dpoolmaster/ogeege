# 02. Каталог функций

Всего выделено **176 функций** в 17 категориях. Приоритеты: P0 — критично для MVP, P1 — важно, P2 — желательно, P3 — отложить.
Статусы: `CONFIRMED`, `OBSERVED`, `NOT_AVAILABLE`, `NOT_TESTED`, `INFERRED`.

---

## AUTH — Аутентификация и доступ

| ID | Функция | Описание / поведение | Priority | Status |
|---|---|---|---|---|
| AUTH-001 | Вход по телефону + код | `id.umschool.net/login`, маска `+7`, страны RU/BY/KZ, кнопка «Получить код». Неверный номер → «Некорректно указан номер». Новый номер → быстрая регистрация после кода | P0 | CONFIRMED (валидация), NOT_TESTED (код) |
| AUTH-002 | Вход по e-mail + паролю | `/login/email`: поля «Почта», «Пароль», «Запомнить меня», «Забыли пароль?». Ошибка: «Невозможно войти с предоставленными учетными данными.» | P0 | CONFIRMED |
| AUTH-003 | Восстановление пароля | `/recover-password`: ввод почты, кнопка «Восстановить» | P1 | CONFIRMED (форма) |
| AUTH-004 | Вход через VK ID | Кнопка «Войти c VK ID» на обеих формах; OAuth-переход | P1 | OBSERVED |
| AUTH-005 | OAuth-callback / обмен токенов | `id` → `nd.umschool.net/oauth-callback?token=...`; токены в `auth-bridge.umschool.net` + fallback в localStorage | P0 | INFERRED/CONFIRMED (токены) |
| AUTH-006 | Смена пароля | Из профиля → `id.umschool.net/change-password`; поля «Новый пароль» (мин. 6), «Повторите пароль», «Подтвердить» | P1 | CONFIRMED |
| AUTH-007 | Подтверждение телефона | Модалка «Подтверждение номера» с кнопкой «Подтвердить» | P1 | CONFIRMED (модалка) |
| AUTH-008 | Выход | Эндпоинт `/api/v2/auth/logout/`; UI-пункт в меню профиля не раскрыт | P1 | OBSERVED |
| AUTH-009 | Сессия/refresh | `/api/v3/auth/refresh/`, `/api/v3/auth/verify/`; авто-refresh access token | P0 | CONFIRMED |
| AUTH-010 | Ограничение доступа | Неавторизованный переход на `/home` → `/login?redirect=/home` | P0 | CONFIRMED |

---

## ONB — Онбординг и цели

| ID | Функция | Описание / поведение | Priority | Status |
|---|---|---|---|---|
| ONB-001 | Выбор класса | `/progress/class-type/selection`: 11–5 классы (в API `classYears`, 8 значений) | P0 | CONFIRMED |
| ONB-002 | Выбор предметов | «Выбери от 3-х до 5-и предметов»; чекбоксы-плитки предметов | P0 | CONFIRMED (текст), NOT_TESTED (сохранение) |
| ONB-003 | Олимпиады | Чекбокс «Участвую в олимпиадах» | P2 | CONFIRMED |
| ONB-004 | Завершение онбординга | `/onboarding/complete`: Telegram-канал преподавателя, наставник, «Перейти на главную» | P1 | CONFIRMED |
| ONB-005 | Нет активных курсов | `/onboarding`: «Нет активных курсов» + «Посмотреть курсы» / «К тренажёру» | P0 | CONFIRMED |
| ONB-006 | Сохранение целей | `POST/PATCH` `/api/umits/individual-program/v1/goals/`, `/api/v2/students/class-types/` | P0 | INFERRED |

---

## PROF — Профиль

| ID | Функция | Описание / поведение | Priority | Status |
|---|---|---|---|---|
| PROF-001 | Просмотр профиля | `/profile?scope=main`: аватар/инициалы, ФИО, контакты, ID, ник, пароль, push | P1 | CONFIRMED |
| PROF-002 | Редактирование телефона | Модалка подтверждения номера; смена — через поддержку | P1 | CONFIRMED (модалка) |
| PROF-003 | E-mail | Отображается; изменение только через чат поддержки | P2 | CONFIRMED (текст) |
| PROF-004 | Привязка VK | Модалка «Подключаем Вконтакте», кнопка «Войти c VK ID» | P3 | CONFIRMED |
| PROF-005 | Ник в чате | Модалка: 3–12 символов, буквы/цифры/`- _ ! ? @ # % & * ( ) + =`, «Сохранить» | P2 | CONFIRMED |
| PROF-006 | Смена пароля | Переход в `id/change-password` | P1 | CONFIRMED |
| PROF-007 | Пуш-уведомления | Тумблер в профиле + модалка разрешения | P2 | CONFIRMED |
| PROF-008 | Вкладка «Документы» | Требует дату рождения; «Не заполнены данные в „Основных сведениях“» | P2 | CONFIRMED |
| PROF-009 | Заполнить данные | CTA «Заполнить данные» (форма даты рождения/доп. сведений) | P2 | OBSERVED |
| PROF-010 | Меню профиля | Кнопка-аватар `profile-menu-trigger-button` | P3 | OBSERVED |

---

## LESS — Занятия и уроки

| ID | Функция | Описание / поведение | Priority | Status |
|---|---|---|---|---|
| LESS-001 | Мои курсы | `/lessons`: карточки продуктов, теги «Доп. курс/ЕГЭ/Школа УМСКУЛ», тариф | P1 | CONFIRMED |
| LESS-002 | Занятия курса | `/lessons/:productTypeId`: «Шкала успеха 0/2», «Основные занятия», фильтры | P1 | CONFIRMED |
| LESS-003 | Модули | `/lessons/:productTypeId/modules/:moduleId` | P1 | OBSERVED |
| LESS-004 | Фильтры занятий | Радио: «Все занятия / Просмотренные / Непросмотренные / Онлайн-помощь / Общая» | P2 | CONFIRMED |
| LESS-005 | Плеер занятия | `/lesson/:id`: видео (Kinescope/YouTube/GCore), таймкоды, описание, материалы, ДЗ | P1 | CONFIRMED |
| LESS-006 | Таймкоды | Раскрывающийся блок, переход по времени | P2 | CONFIRMED (API) |
| LESS-007 | Материалы занятия | `/api/lessons/:id/materials/`; пусто в демо | P2 | CONFIRMED (empty) |
| LESS-008 | ДЗ к занятию | `/api/v2/lessons/:id/homework/`: count_task, solve_interval, earned_exp, max_exp | P1 | CONFIRMED |
| LESS-009 | Мини-тест после занятия | `/api/mini-tests/lessons/:id/status/`, статус `NONE` в демо | P2 | CONFIRMED (API) |
| LESS-010 | Оценка занятия | «Оцени занятие», рейтинг занятия | P2 | OBSERVED |
| LESS-011 | Просмотр/прогресс | `viewed`, `lesson_passed`, «Посмотри 15 минут урока» для зачёта | P1 | CONFIRMED/INFERRED |
| LESS-012 | Доступ по ссылке | `/lesson/access/:lessonHash` | P2 | OBSERVED |
| LESS-013 | Live/запись | `is_live`, `lesson_is_live_right_now`, «Занятия в записи» | P2 | OBSERVED |

---

## HW — Домашние работы

| ID | Функция | Описание / поведение | Priority | Status |
|---|---|---|---|---|
| HW-001 | Список ДЗ | `/homework/list?tab=HOMEWORK&month=11`; вкладки «Домашние работы / Индивидуальные ДЗ» | P1 | CONFIRMED |
| HW-002 | Фильтры ДЗ | Месяц, «Все статусы», «Все типы» (Обычное/Работа над ошибками/Повторение), сброс | P2 | CONFIRMED |
| HW-003 | Пустой список | «По твоему запросу нет результатов. Попробуй изменить условия поиска.» + «Сбросить» | P1 | CONFIRMED |
| HW-004 | Старт ДЗ | `POST /api/v2/homework/submissions/:id/start/`; запуск таймера | P1 | CONFIRMED |
| HW-005 | Типы заданий | TEST (варианты), TEXT (rich-редактор), WORD (краткий ответ), AUTO/NO_CHECK/CURATOR | P1 | CONFIRMED |
| HW-006 | Ответ/пропуск | «Ответить» / «Пропустить»; пустой ответ → «Ответ не может быть пустым» | P1 | CONFIRMED |
| HW-007 | Rich-редактор | Жирный/курсив/индексы/выравнивание/шрифт/изображение/файл/ссылка/видео/MathType/ChemType | P2 | CONFIRMED |
| HW-008 | Навигация по заданиям | Номера 1..N, таймер, «К списку домашних заданий», «Версия для печати» | P1 | CONFIRMED |
| HW-009 | Завершение | «Завершить», подтверждение, автоотправка по таймеру | P1 | CONFIRMED |
| HW-010 | Отказ | «Отказаться от выполнения» + подтверждение; фильтр «Отказ» | P2 | CONFIRMED (кнопка) |
| HW-011 | Результат | `/completed`: баллы, время, «6 заданий решено / 0 правильно / 1 неправильные», «0 ОП», шкала до уровня | P1 | CONFIRMED |
| HW-012 | Работа над ошибками | `/work-on-bugs`: разбор заданий, «Получено баллов», «Далее» | P1 | CONFIRMED |
| HW-013 | Печать | `/print`: все задания без интерфейса | P2 | CONFIRMED |
| HW-014 | Оценка ДЗ | 1–5 + причины (например, «Мало практики», «Слишком сложно») | P3 | CONFIRMED (API) |
| HW-015 | Проверка куратором/Умкой | Статусы CHECKED_BY_SITE / CHECKED_BY_CURATOR / CHECKED_BY_NEURO / CHECK_BY_NE_AND_CU | P2 | INFERRED |
| HW-016 | Запрос доп. проверки | «Запросить доп. проверку», «Проверку проведёт другой куратор» | P3 | OBSERVED |
| HW-017 | Сообщить об ошибке | «Если ты нашел ошибку в этом задании, сообщи нам» | P3 | OBSERVED |
| HW-018 | Генерация ДЗ | `/homework/generated/:submissionId` (шаблоны, KIM-номера, подтемы, уровни) | P2 | OBSERVED (API) |

---

## CAL — Расписание

| ID | Функция | Описание / поведение | Priority | Status |
|---|---|---|---|---|
| CAL-001 | Недельный календарь | ПН–ВС, даты, «Сегодня», заголовок «Ноябрь 2026» | P1 | CONFIRMED |
| CAL-002 | Режимы | URL `?type=lesson` (переключение на ДЗ — «Переключай расписание, чтобы увидеть дедлайны») | P2 | CONFIRMED |
| CAL-003 | Навигация по месяцам | `month`/`year` в URL; клик по месяцу открывает выбор | P2 | CONFIRMED |
| CAL-004 | Добавить предмет | Кнопка ведёт в `/catalog` | P1 | CONFIRMED |
| CAL-005 | Событие дня | Клик по дню/предмету → время и переход к занятию (в демо событий нет) | P2 | INFERRED |
| CAL-006 | Часовой пояс | Время отображается «МСК», выбора TZ нет | P2 | CONFIRMED |
| CAL-007 | API | `/api/v1/timetable/lessons/?year=&month=`, `/timetable/homeworks/`, `/class-types/filters/`, `/core/class-types/colors/` | P1 | CONFIRMED |

---

## TRAIN — Тренажёр и аналитика

| ID | Функция | Описание / поведение | Priority | Status |
|---|---|---|---|---|
| TRAIN-001 | Дашборд тренажёра | `/progress`: предметы, предсказание балла, вкладки «График/Темы/Номера КИМ/Пробные варианты» | P0 | CONFIRMED |
| TRAIN-002 | Выбор предметов | «Изменить предметы» → `/progress/class-type/selection` | P1 | CONFIRMED |
| TRAIN-003 | Настройки тренировки | `/training/settings`: «Карточки/Заучивание», статусы, избранное, время, количество, скрыть ответы | P0 | CONFIRMED |
| TRAIN-004 | Статусы карточек | Новое / Изучаю / Знаю / Повторяю / Только избранное | P1 | CONFIRMED |
| TRAIN-005 | Сессия карточек | `/training/cards`: вопрос, ответ, «Не знаю/Знаю», прогресс `0/42835`, счётчики, «Завершить» | P0 | CONFIRMED |
| TRAIN-006 | Клавиатура | «Используй стрелки ← →» | P3 | CONFIRMED (текст) |
| TRAIN-007 | Результат тренировки | «Тренировка завершена», +50 ОП, список карточек по статусам | P1 | CONFIRMED |
| TRAIN-008 | Аналитика по темам | Список тем и подтем, поиск по названию, «График» (disabled без данных) | P2 | CONFIRMED |
| TRAIN-009 | Номера КИМ | Вкладка с заданиями по номерам КИМ | P2 | OBSERVED |
| TRAIN-010 | Пробные варианты | Empty state «Пробных вариантов пока нет» + «Сгенерировать» | P2 | CONFIRMED |
| TRAIN-011 | Генератор варианта | `POST /homework/generated-homeworks/generation/*` (KIM, подтемы, уровни) | P2 | OBSERVED (API) |
| TRAIN-012 | Уровень знаний | «ИИ анализирует…», шкала 100%, «Не хватает данных» | P2 | INFERRED |
| TRAIN-013 | Тест уровня | `/level-test/:testId`: задание+решение, «Да/Нет», «Завершить», предупреждение о незавершённых | P2 | CONFIRMED (ошибка теста) |
| TRAIN-014 | API карточек | `/api/cards/count/`, `/api/cards/`, `/cards/training/start|answer|finish|result/` | P0 | CONFIRMED |

---

## CAT — Каталог и продукты

| ID | Функция | Описание / поведение | Priority | Status |
|---|---|---|---|---|
| CAT-001 | Хаб магазина | `/catalog`: 5 карточек (Основной курс, Школьная подготовка, Олимпиады, Умскул×Репетиторы, Курсы) | P0 | CONFIRMED |
| CAT-002 | Список каталога | `/catalog/:code`: фильтры «Все классы/предметы/учителя», карточки продуктов, «Подробнее/Выбрать» | P0 | CONFIRMED |
| CAT-003 | Фильтры | Класс, предмет, преподаватель, язык; сброс | P1 | CONFIRMED |
| CAT-004 | Пагинация/догрузка | «Загрузить ещё», limit/offset | P1 | CONFIRMED |
| CAT-005 | Карточка продукта | `/catalog/:code/:productId`: описание, даты, преподаватель, тарифы, «Записаться/Выбрать» | P0 | CONFIRMED |
| CAT-006 | Тарифы | Стандарт / Про / Премиум с ценой и списком фич; сравнение | P0 | CONFIRMED |
| CAT-007 | Добавление в корзину | «Выбрать» → `POST /api/v2/cart/` | P0 | CONFIRMED |
| CAT-008 | Комплект | `/catalog/:code/bundles/configure`: шаг 1 класс+предметы (2–8/2–4), шаг 2 преподаватели, срок, тариф | P1 | CONFIRMED |
| CAT-009 | Публичный каталог | `/courses/`, `/ege/`, `/oge/`, `/srednyaya-shkola/`, `/olimpiady/` с фильтрами и ценами | P1 | CONFIRMED |
| CAT-010 | Хлебные крошки/SEO | Публичные хлебные крошки, JSON-LD (на предметных страницах) | P2 | OBSERVED |

---

## CART — Корзина

| ID | Функция | Описание / поведение | Priority | Status |
|---|---|---|---|---|
| CART-001 | Просмотр корзины | `/cart?paymentMethod=card`; список позиций, суммы, скидки, рекомендации | P0 | CONFIRMED |
| CART-002 | Пустая корзина | «В корзине пока пусто» + «Перейти в магазин» | P0 | CONFIRMED |
| CART-003 | Специальная цена | Автодобавление предыдущего месяца со скидкой (`optional_gifts`) | P1 | CONFIRMED |
| CART-004 | Промокод | «У меня есть промокод» → поле; ошибка «Некорректный промокод.»; «Промокоды не суммируются» | P1 | CONFIRMED |
| CART-005 | Реферальная скидка | `referral_program` в корзине; «Скидка по реферальной программе» | P2 | CONFIRMED (API) |
| CART-006 | Ссылка на оплату | «Оплата по ссылке» → `/cart-owned-by-another?hash=...` | P1 | CONFIRMED |
| CART-007 | Очистка | «Удалить всё» и `POST /api/v2/cart/clear/` | P2 | CONFIRMED |
| CART-008 | Счётчик | `/api/v2/cart/count/`, бейдж в шапке | P1 | CONFIRMED |
| CART-009 | Кол-во/состав | `POST/PATCH /api/v2/cart/`, `gifts`, `bundles` | P1 | CONFIRMED (API) |
| CART-010 | Рекомендации | `/api/products/recommendations/`, «Добавить» | P2 | CONFIRMED |

---

## PAY — Платежи и подписки

| ID | Функция | Описание / поведение | Priority | Status |
|---|---|---|---|---|
| PAY-001 | Checkout | `pay.umschool.net/payment/:uuid`: контакты, способ оплаты, сумма, «Оплатить» | P0 | CONFIRMED (без оплаты) |
| PAY-002 | Способы оплаты | СБП, карта, рассрочка, Яндекс Сплит, T-Pay, Яндекс Пэй, реквизиты (физ/юр), цифровой рубль | P0 | CONFIRMED |
| PAY-003 | Скидка за способ оплаты | Показ `−20%` / `−18%`, «Скидка за способ оплаты» | P1 | CONFIRMED |
| PAY-004 | Контактные данные | «Данные обязательны к заполнению», поля заказчика/ученика | P0 | CONFIRMED (текст) |
| PAY-005 | Оплата по реквизитам | Реквизиты, QR, «Прикрепить чек», проверка 30 минут | P2 | CONFIRMED (страница способов) |
| PAY-006 | Рассрочка | Сбербанк / Т-Банк, сумма > 5 000 ₽, 3/6/12 мес., 18–70 лет | P2 | CONFIRMED (публичная страница) |
| PAY-007 | Успешная оплата | `/payment/success/:transactionUuid` | P2 | OBSERVED |
| PAY-008 | Подписки | `/subscriptions`: активные/неактивные, «История списаний» | P1 | CONFIRMED (empty) |
| PAY-009 | Продление | `/subscriptions/:id/renewal`, manual prolong | P1 | NOT_AVAILABLE |
| PAY-010 | Улучшение | `/subscriptions/:id/upgrade`, manual upgrade | P2 | NOT_AVAILABLE |
| PAY-011 | Автопродление | Вкл/выкл, причины отказа, привязка карты | P1 | OBSERVED/INFERRED |
| PAY-012 | Способы оплаты | `/api/payment-methods/`, привязка/удаление | P1 | OBSERVED |
| PAY-013 | Причины отключения | `deactivate/reasons/`: manual_payment, bad_date, low_progress, not_ready, bad_fit, other | P2 | CONFIRMED |
| PAY-014 | Документы/чеки | «Документы и чеки доступны для последней оплаты» | P2 | OBSERVED |

---

## GAME — Геймификация

| ID | Функция | Описание / поведение | Priority | Status |
|---|---|---|---|---|
| GAME-001 | Уровни | 61 уровень, EXP 0…100 000, награды умкоинами/медалями/ключами | P2 | CONFIRMED |
| GAME-002 | Баланс | `/api/v2/students/student-balance/`: exp, umcoins, keys, medals, bonuses, ambassadorBonuses; лимиты 800/300 | P1 | CONFIRMED |
| GAME-003 | Достижения | 8 ачивок по 5–8 уровней (бронза…космос), медали | P2 | CONFIRMED |
| GAME-004 | Ежедневная серия | `daily-strike`: strike_days, freeze_days, next_reward, week_statuses; заморозка 5 умкоинов/30 медалей | P2 | CONFIRMED |
| GAME-005 | Индивидуальные задания | Ежедневные/недельные/месячные: «Реши 5 заданий», «Реши 20», «7 ДЗ на 60+» | P1 | CONFIRMED |
| GAME-006 | Купоны | Обмен умкоинов: 50 → 150, 150 → 500 | P2 | CONFIRMED (API) |
| GAME-007 | Мерч | Ссылка на `store.umschool.net` | P3 | CONFIRMED |
| GAME-008 | История баланса | `balance-history`: начисления/списания, тип, урок | P2 | CONFIRMED |
| GAME-009 | Кастомизация чата | Стикеры, цвета текста за валюту (`/gamification/chat/styling/*`) | P3 | OBSERVED (API) |

---

## REF — Реферальная программа

| ID | Функция | Описание / поведение | Priority | Status |
|---|---|---|---|---|
| REF-001 | Страница приглашения | `/referral/invite`: ссылка, «Скопировать», «Показать QR-код», «Поделиться» | P1 | CONFIRMED |
| REF-002 | Условия | Чекбокс «Я принимаю условия программы лояльности и публичной оферты» | P1 | CONFIRMED |
| REF-003 | Вознаграждения | Ноябрьский поток: 1 мес — 2 000 бонусов тебе / 1 000 ₽ другу; 1 год — 3 500 / 2 000 ₽ | P1 | CONFIRMED |
| REF-004 | Баланс | bonuses, ambassador_bonuses, invited_friends_count, used_bonuses | P1 | CONFIRMED |
| REF-005 | Вывод | «Вывести на карту», только на личную карту, ФИО должны совпадать | P2 | OBSERVED |
| REF-006 | История | `/referral/history`: дата, кол-во, тип, продукт/телефон, статус | P2 | CONFIRMED (empty) |
| REF-007 | FAQ | `/api/core/faq/?category=REFERRAL` + аккордеон | P2 | CONFIRMED |
| REF-008 | Обратная связь | `/api/v1/referral/feedback/` опрос о программе | P3 | CONFIRMED (API) |
| REF-009 | Ambassador | `ambassadorBonuses`, `/referral/pay-ambassador/`, документы | P3 | OBSERVED |

---

## MSG — Сообщения/куратор

| ID | Функция | Описание / поведение | Priority | Status |
|---|---|---|---|---|
| MSG-001 | Список диалогов | `/chat`: «Диалогов нет», «Выбери предмет» | P1 | CONFIRMED (empty) |
| MSG-002 | Создание диалога | Выбор предмета (в демо нет доступных) | P1 | INFERRED |
| MSG-003 | Отправка сообщений | `POST /messenger/v1/student/topics/...` | P1 | NOT_TESTED |
| MSG-004 | Поиск сообщений | `/messenger/v1/student/messages/search/`, «Найдено: N» | P2 | OBSERVED (API) |
| MSG-005 | Непрочитанные | `topics/unread-count/`, бейдж | P1 | CONFIRMED |
| MSG-006 | System events | `/messenger/v1/student/system-event/` | P2 | OBSERVED |
| MSG-007 | Обратная связь | `/topics/feedback/` | P2 | OBSERVED |
| MSG-008 | Передача куратору | «Передать диалог куратору», «Случайному куратору» | P3 | OBSERVED |
| MSG-009 | Режим работы | «Сейчас не рабочее время… Пн-пт 13:00–23:00, Сб-вс 11:00–23:00 МСК» | P2 | CONFIRMED (текст) |
| MSG-010 | Real-time | Centrifugo `wss://chat-centrifugo.umschool.net/connection/websocket` | P2 | OBSERVED |

---

## NOTIF — Уведомления

| ID | Функция | Описание / поведение | Priority | Status |
|---|---|---|---|---|
| NOTIF-001 | Push-модалка | «Хотите получать уведомления?» + «Включить уведомления» / «Не сейчас» | P1 | CONFIRMED |
| NOTIF-002 | Web push | `/api/v2/core/webpush/subscribe|unsubscribe/`, service worker `/webpush/service-worker.js` | P1 | CONFIRMED (API) |
| NOTIF-003 | Настройка в профиле | Тумблер «Пуш-уведомления» | P1 | CONFIRMED |
| NOTIF-004 | Mindbox webpush | `/api/v2/core/mindbox/webpush/`, `/mindboxwebpush/service-worker.js` | P2 | CONFIRMED (код) |
| NOTIF-005 | E-mail/SMS | Подтверждение, восстановление пароля; провайдер не раскрыт | P1 | INFERRED |
| NOTIF-006 | Системные модалки | `system-modals`: подтверждение телефона, согласие ПД, реклама | P1 | CONFIRMED |
| NOTIF-007 | Уведомления о ДЗ/занятиях | Обещаны в push-модалке и настройках | P1 | INFERRED |

---

## DOC — Документы

| ID | Функция | Описание / поведение | Priority | Status |
|---|---|---|---|---|
| DOC-001 | Общие документы | `/documents`: «Сведения об образовательной организации» | P2 | CONFIRMED (empty) |
| DOC-002 | Документы по продуктам | `students/documents/product-types/` | P2 | CONFIRMED (empty) |
| DOC-003 | Личные документы | `personal_data`, `loyalty_program`, `apartment_lottery`, `ambassador_program` | P2 | CONFIRMED (null) |
| DOC-004 | Публичные документы | Оферта, политика, пользовательское соглашение, согласие на рассылки, лицензия | P1 | CONFIRMED |
| DOC-005 | Чеки | Доступны за последнюю оплату; ранние — через поддержку | P2 | OBSERVED |

---

## PUB — Публичный сайт и бесплатные инструменты

| ID | Функция | Описание / поведение | Priority | Status |
|---|---|---|---|---|
| PUB-001 | Главная | Hero, соцдоказательства, курсы, отзывы, FAQ, лид-форма | P0 | CONFIRMED |
| PUB-002 | Каталог | `/courses/`, фильтры (курсы/классы/предметы/цели/преподаватели), карточки с ценой/мес и за курс | P0 | CONFIRMED |
| PUB-003 | Предметные лендинги | `/ege/{...}`, `/oge/{...}`, `/srednyaya-shkola/{...}`, `/olimpiady/{...}` | P1 | CONFIRMED |
| PUB-004 | Преподаватели | `/tutor/`: 60 курсов, фильтры, профили, статистика | P1 | CONFIRMED |
| PUB-005 | Отзывы | `/reviews/`: фильтры «Мой класс/Предмет/Преподаватели», пагинация 97 стр., «Читать полностью» | P2 | CONFIRMED |
| PUB-006 | FAQ | `/faq/`: категории «Другое/Процесс обучения/Оплата/Поддержка» | P2 | CONFIRMED |
| PUB-007 | Контакты | Адрес Казань, ОГРН/ИНН, телефоны, почта, часы | P1 | CONFIRMED |
| PUB-008 | Способы оплаты | Карта, СБП, Яндекс Пэй/T-Pay, рассрочка Сбер/Т-Банк, реквизиты | P1 | CONFIRMED |
| PUB-009 | Промокоды | `/coupons/free/`: сезонные акции и бесплатные курсы, таймер, «1 шт.» | P2 | CONFIRMED |
| PUB-010 | Банк заданий | 76 158 заданий, фильтры (экзамен, предмет, раздел, тема, часть, КИМ, источник, нерешённые, избранные) | P1 | CONFIRMED (API) |
| PUB-011 | Пробные варианты | Список демоверсий/вариантов, «Выполнить» | P1 | CONFIRMED |
| PUB-012 | Интерактивные тренажёры | Каталог игровых тренажёров по предметам | P2 | CONFIRMED |
| PUB-013 | Эмулятор КЕГЭ | `/free/emulyator-kege/` — HTTP 500 «temporarily unavailable» | P2 | CONFIRMED (ошибка) |
| PUB-014 | Родителям | Отдельный лендинг, соцдоказательства, лид-форма | P1 | CONFIRMED |
| PUB-015 | О школе | История 2016→, миссия, шаги обучения | P3 | CONFIRMED |
| PUB-016 | Лид-формы | `full_name`, `phone`, `email` (в части форм), чекбокс согласия, honeypot `additional-field-*` | P0 | CONFIRMED |
| PUB-017 | Журнал/Учебник | `/journal/`, `/library/` (внешние разделы) | P3 | OBSERVED |
| PUB-018 | Карта сайта | `/sitemap/` | P3 | OBSERVED |
| PUB-019 | Юридические документы | Оферта, политика, согласие, лицензия, реестр ПО | P1 | CONFIRMED |
| PUB-020 | Партнёры/сотрудничество | `landing.umschool.net/partners`, `/collaboration`, `cpa.umschool.net` | P3 | OBSERVED |

---

## ADMIN — Администрирование (недоступно)

| ID | Функция | Описание | Priority | Status |
|---|---|---|---|---|
| ADM-001 | Роли и доступ | 16+ ролей из enum; разграничение по `meta.roles` | P1 | OBSERVED (код) |
| ADM-002 | Управление курсами/уроками | `lesson_edit`, `lesson_deleting`, `lessons_are_being_prepared` | P1 | NOT_AVAILABLE |
| ADM-003 | Проверка ДЗ | Кураторские комментарии, критерии, апелляции | P1 | NOT_AVAILABLE |
| ADM-004 | Менеджмент подписок/платежей | Manual prolong/upgrade, pay inspector | P2 | NOT_AVAILABLE |
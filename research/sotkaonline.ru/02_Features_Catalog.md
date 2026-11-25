# 02. Каталог функций

Формат: Feature ID, название, категория, раздел, описание, входы/действия/выходы, состояния/ограничения, статус.

Статусы: **CONFIRMED** — проверено в UI; **OBSERVED** — найдено в DOM/JS/API, сценарий не пройден; **INFERRED** — восстановлено по коду; **NOT_AVAILABLE** — нет доступа; **NOT_TESTED** — не проверялось.

---

## AUTH — Аутентификация

### AUTH-001. Вход по телефону/почте с кодом
- **Раздел:** `platform.sotkaonline.ru/auth`
- **Описание:** вход для существующих клиентов по номеру телефона или российской почте; отправка кода по SMS/письму; ввод кода.
- **Входы:** телефон или email, чекбокс «Сохранить вход».
- **Действия:** выбор таба «Телефон»/«Почта» → ввод → «Продолжить» → ввод кода → вход.
- **Выходы:** сессия, редирект в ЛК; ошибки.
- **Валидация (CONFIRMED):**
  - пустое поле → «Это обязательное поле»;
  - невалидная почта → «Авторизация по этой почте невозможна. Войди по номеру…»;
  - несуществующий пользователь → «Пользователь не найден.» (HTTP 422 `send-verify-code`).
- **Ограничение:** регистрация через эту форму недоступна.
- **API:** `POST auth.sotkaonline.ru/api/v2/send-verify-code` `{login, new_platform}`; `POST /api/v1/sign-in/{type}`.
- **Статус:** CONFIRMED (форма, валидации); вход с кодом NOT_TESTED.

### AUTH-002. «Я тут впервые» — развилка
- **Описание:** два пути: «Хочу больше узнать о платформе» (лид-форма) и «Я всё знаю и хочу оплатить обучение» (переход в магазин).
- **Выходы:** лид-форма (класс, телефон) или `sotkaonline.ru/sotkastore`.
- **Статус:** CONFIRMED.

### AUTH-003. Сохранение входа
- **Описание:** чекбокс `accessSave`; INFERRED: долгоживущая сессия/refresh-token.
- **Статус:** OBSERVED.

### AUTH-004. Выход
- **API:** `POST /logout`, `GET /auth/logout`; очистка токенов.
- **Статус:** INFERRED.

### AUTH-005. Смена/подтверждение email
- **API:** `POST /profile/change-email/send-code`, `POST /profile/change-email/confirm`.
- **Статус:** INFERRED.

### AUTH-006. Отзыв токенов
- **API:** `GET /profile/revoke-tokens`.
- **Статус:** INFERRED.

### AUTH-007. Вход преподавателя
- **API:** `POST /teacher/sign-in`, `/teacher/forgot-password`, `/teacher/password-reset`, `/teacher/email/verify`, `/teacher/email/resend`, `/teacher/register`.
- **Статус:** INFERRED.

---

## REG — Регистрация и онбординг

### REG-001. Создание аккаунта при покупке
- **Описание:** отдельной публичной регистрации нет; аккаунт создаётся в процессе оформления доступа (имя, телефон, email, VK, промокод, согласия).
- **Подтверждение:** `send-verify-code` для нового email → 422 «Пользователь не найден»; в магазине FAQ: «Пользователь с данной почтой привязан к другому телефону».
- **Статус:** CONFIRMED (отсутствие самозаписи), INFERRED (момент создания).

### REG-002. Онбординг-опросник (10 шагов)
- **Маршрут:** `/register`; состояние `register` с полями: `first_name`, `knowledge_poll`, `email`, `phone_number`, `vk`, `telegram`, `goals_poll[]`, `trials_data[]`, `purposes[]`, `favorite_courses[]`, `focus_courses[]`.
- **Шаги (OBSERVED):** имя → цель → фокусные предметы → баллы пробников → контакты → настройка программы.
- **Валидация:** балл 0..max («Значение не должно превышать N»), обязательные имя/почта/телефон, формат даты.
- **Статус:** OBSERVED/INFERRED.

### REG-003. Персональный план
- **Описание:** по цели и баллам формируется интенсивность программы; фокусные предметы идут первыми.
- **API:** `POST /profile/purposes`, `POST /profile/first-visit`, `POST /profile/first-visit/courses`.
- **Статус:** INFERRED.

### REG-004. Trial-доступ
- **Признаки:** `is_trial`, `/trial`, `isMiddleTrial`, заглушки «Подписка…».
- **API:** `GET /trial` (пробники), `POST /promotions/demo-access`, `/verify`, `/courses` (демо-доступ).
- **Статус:** OBSERVED; как получить — UNKNOWN.

---

## STORE — Магазин и тарифы

### STORE-001. Конфигуратор класса
- **UI:** «1-8 классы» / «9-11 классы» → класс 11/10/9 (для ЕГЭ/ОГЭ).
- **API:** `GET /store/sotka-grades`.
- **Статус:** CONFIRMED.

### STORE-002. Выбор предметов (1–4)
- **UI:** карточки предметов с пометкой `*` — «можно выбрать учителя»; счётчик «Выбрано N из 4»; «Далее» disabled при 0.
- **API:** `GET /store/sotka-courses?grade=&year_type=`.
- **Статус:** CONFIRMED (UI), выбор карточки OBSERVED.

### STORE-003. Выбор тарифа
- **Тарифы:** Базовый, Стандарт, Про, Про+, Бюджет.
- **Периоды:** месяц / учебный год.
- **API:** `GET /store/sotka-products?grade=&special=0`.
- **Статус:** CONFIRMED (цены/состав).

### STORE-004. Оформление (checkout)
- **Поля (OBSERVED):** имя, телефон, email, ссылка VK, промокод `sotka_promo`, согласия (ПДн/оферта), данные ребёнка/родителя, код подтверждения.
- **Действия:** проверка промокода, выбор оплаты/рассрочки, подтверждение кода.
- **API:** `/promo-code/check`, `/store/checkAddCourses`, `/store/addCoursesToAccesses`, `/store/renew`, `/store/updateAccess`, `/store/checkRenew`, `/store/checkUpdate`.
- **Статус:** OBSERVED (поля), оплата NOT_TESTED.

### STORE-005. Промокоды
- **API:** `POST /promo-code/check`.
- **Статус:** INFERRED.

### STORE-006. Рассрочка/оплата частями
- **Провайдер:** CloudPayments (`widget.cloudpayments.ru`), банки-партнёры (FAQ).
- **Статус:** OBSERVED (подключение скрипта), NOT_TESTED.

### STORE-007. Тариф «Бюджет» (0 ₽)
- **Описание:** 0 ₽ сейчас, оплата при поступлении на бюджет; заявка.
- **Статус:** CONFIRMED (текст), условия — UNKNOWN.

---

## MAIN — Главная и план

### MAIN-001. Главная страница ЛК
- **Маршрут:** `/mainpage`.
- **Содержимое (OBSERVED):** приветствие, прогресс теорий, ближайшие эфиры, «Решить пробник», «Сделать домашку», реферальный блок, навигация по предметам.
- **API:** `GET /mainpage`, `GET /mainpage/statistic-by-month`, `GET /quotes`.
- **Статус:** INFERRED.

### PLAN-001. План на сегодня
- **Маршрут:** `/daily-plan`.
- **Содержимое:** «На сегодня по плану», теория/практика, «Осталось с прошлых дней», «Пропущенное», «Работа над ошибками», прогресс «Сделано N из M уроков».
- **API:** `GET /study-plan?course_id=`, `GET /study-plan/missed?course_id=`.
- **Статус:** INFERRED.

---

## COURSE — Курсы и уроки

### COURSE-001. Мои предметы
- **Маршрут:** `/courses`.
- **Содержимое:** карточки оплаченных курсов, прогресс, переход в программу.
- **API:** `GET /course`, `GET /course?with_progress_bar=true`, `GET /courses`.
- **Статус:** INFERRED.

### COURSE-002. Программа курса
- **Маршрут:** `/courses/:courseId`.
- **Содержимое:** блоки (от 4 до 9), темы, уроки, зачёты, вводные/основные занятия, доп. обучение, профориентация, смена преподавателя, чат комьюнити.
- **API:** `GET /course/{id}`, `/months`, `/theme`, `/lessons/by-month`, `/lessons/calendar`, `/user_info`, `/statistic`, `/child_courses`, `/change_courses`.
- **Статус:** INFERRED.

### COURSE-003. Смена преподавателя
- **Действие:** `/course/{id}/change_course`, `/change_courses`, `/chose_child_course`; модалка `change-teacher_modal`.
- **Ограничение:** только курсы с `can_change_teacher` / пометкой `*`.
- **Статус:** INFERRED.

### COURSE-004. Смена/выбор дочернего курса
- **API:** `GET /course/{id}/child_courses`, `POST /course/{id}/chose_child_course`.
- **Статус:** INFERRED.

### LESSON-001. Урок
- **Маршрут:** `/courses/:courseId/:lessonId`.
- **Содержимое (OBSERVED):** видео, запись эфира, статья/конспект, таймкоды, тест, домашка, «Домашка Про», практика, «В этом уроке», «Содержание», «Пройди все занятия», «Индивидуальный зачёт», «Начать зачёт», «Предыдущий/Следующий блок».
- **Действия:** просмотр видео (учёт глубины), чтение статьи, тест, переход к ДЗ.
- **API:** `GET /course/{id}/lesson/{lessonId}`, `/article`, `/home_work`; `POST /lesson/{lessonId}/video-depth`, `/life-depth`.
- **Статус:** INFERRED.

### LESSON-002. Зачёт
- **Описание:** индивидуальный зачёт по блоку/теме, «Зачёт сдан», «Начать зачёт».
- **API:** `GET /course/{id}/block/{blockId}/final-test`.
- **Статус:** INFERRED.

### LESSON-003. Прогресс теории/практики
- **Правило (OBSERVED):** теория засчитывается при просмотре видеоурока; практика/онлайн-занятие — при выполненной домашке.
- **Статус:** INFERRED (тексты в UI).

---

## HW — Домашние задания

### HW-001. Список домашек
- **Маршрут:** `/homework`.
- **Фильтры:** Все / К выполнению / Выполнено; типы «Домашка», «Пробники».
- **API:** `GET /homeworks`, `/homeworks/months`, `/homeworks/subjects`.
- **Статус:** INFERRED.

### HW-002. Выполнение ДЗ
- **Маршрут:** `/homework/:hwId`.
- **UI:** «Задание №N», «КИМ №», «Обязательный/Необязательный вопрос», «Сложность», навигация «Назад/Далее», «Завершить», «Проверить».
- **Состояния:** «Сохранение…», «Ответ сохранён», «Ошибка при сохранении», «Ошибка завершения», «Не удалось завершить…».
- **API:** `GET /homework/{id}`, `POST /homework/{id}/save-answer`, `/save-instant-answer`, `/save-custom-answer`, `POST /homework/{id}/complete`, `/custom-complete`.
- **Статус:** INFERRED.

### HW-003. Режим min/max
- **Описание:** адаптивное ДЗ: минимальный (обязательный) и максимальный (дополнительный) объём; выбор режима.
- **Статус:** INFERRED (текст тарифов).

### HW-004. Автопроверка тестов
- **Правило:** тесты проверяются автоматически; развёрнутые ответы — вручную.
- **Статус:** INFERRED.

### HW-005. Проверка наставником
- **Действия:** «Отправить наставнику», «Проверка займёт время, но будет с экспертной оценкой», «Нужна помощь наставника?», «Обращений к наставнику».
- **API:** `POST /study/review`, `POST /study/review/to-mentor`.
- **Статус:** INFERRED.

### HW-006. Самопроверка
- **Маршрут:** `/selfcheck/:hwId`.
- **UI:** «Оцени свои решения», «Подвести итоги», «Показать моё решение», «Критерии оценивания», ползунок балла, «В этом задании выставляется N баллов».
- **Статус:** INFERRED.

### HW-007. Апелляция
- **Действие:** «Подать апелляцию на оценку задания», комментарий, статус апелляции.
- **API:** `POST /homework/{id}/answers/{answerId}/appeal`.
- **Статус:** INFERRED.

### HW-008. Тематические ДЗ (tagHomework)
- **Маршрут:** `/tagHomework/:hwId`.
- **API:** `GET /tagHomework/{id}?tag_id=`, `POST /tagHomework/{id}/result?tag=`.
- **Статус:** INFERRED.

### HW-009. Сброс ответов
- **API:** `POST /homework/{id}/answers/reset`, `POST /homework/{id}/answers/tag-reset?tag_id=`.
- **Статус:** INFERRED.

---

## SAMPLE — Пробники и контрольные

### SAMPLE-001. Список пробников
- **Маршрут:** `/samples`.
- **UI:** «Пробники»/«Контрольные», «Домашка к уроку от …», «Пройти пробник», «Решить пробник».
- **API:** `GET /trial?month=`.
- **Статус:** INFERRED.

### SAMPLE-002. Прохождение пробника
- **Описание:** полноценный экзамен, ручная проверка экспертами 90+.
- **Статус:** INFERRED.

### SAMPLE-003. Результат
- **API:** `GET /homework/{id}/result`, `/tagHomework/{id}/result`.
- **Статус:** INFERRED.

---

## TASK — Банк заданий

### TASK-001. Публичный банк заданий
- **Маршрут:** `/bank-zadaniy`.
- **UI (CONFIRMED):** табы «я сдаю ЕГЭ/ОГЭ», выбор предмета, выбор темы, кнопка «Решать».
- **Данные:** 2 курса на экзамен (профильная математика, русский), темы = `active_tag_homeworks`.
- **Статус:** CONFIRMED.

### TASK-002. Решение варианта
- **UI (CONFIRMED):** заголовок темы, «Вариант №…», навигация 1..4, «Задание №1», поле «Ответ», кнопка «Ответить», после ответа — «Правильный ответ N», «Продолжить», прогресс «1/4».
- **Особенность:** правильные ответы приходят в `GET /public-task-bank/homeworks/{id}` и сравнение выполняется на клиенте.
- **Статус:** CONFIRMED.

### TASK-003. Банк заданий в ЛК
- **Маршрут:** `/tasksStorage`.
- **UI:** «Выбери тему», «Номер КИМ», «Настрой задания под себя», «История тестов», «Тест от …», «Решённых вопросов банка заданий».
- **API:** `GET /tagsBank`, `GET /schoolbooks?course_id=`, `GET /study/theme/{id}/files`.
- **Статус:** INFERRED.

---

## STAT — Статистика

### STAT-001. Аналитика ученика
- **Маршрут:** `/statistics`.
- **UI:** «Аналитический центр», выбор предмета, метрики: занятия, домашки, домашки стандарт, пробники, контрольные, «на сегодня инсайтов нет», «Оформить подписку».
- **API:** `GET /statistics`, `/statistics/{id}`, `/profile/progress/today`.
- **Статус:** INFERRED.

### STAT-002. Публичный отчёт
- **Маршрут:** `/statistics/:id` (без авторизации).
- **Поведение (CONFIRMED):** страница открывается, показывает «Статистика», период, «Активность», «Успеваемость», при неверном id — «Произошла ошибка».
- **Статус:** CONFIRMED (каркас), данные NOT_TESTED.

### STAT-003. Родительская статистика
- **API:** `GET /parent/statistic/courses`, `/parent/statistic/courses/lessons`.
- **Статус:** INFERRED.

---

## CAL — Расписание

### CAL-001. Расписание
- **Маршруты:** `/calendar`, `/timetable`.
- **UI:** «Расписание», «Календарь дня», «Сегодня», «Предыдущий/Следующий день/неделя/месяц», «Список», фильтр «Все предметы», «Онлайн-занятие», «Практика», «Видео и домашка», «Пройдено N из M», «Пройди теорию, чтобы открыть практику», «Здесь только онлайн-занятия — по твоему часовому поясу».
- **API:** `GET /calendar`, `/calendar/month`, `/course/{id}/lessons/calendar`, `/lessons/by-month`.
- **Статус:** INFERRED.

### CAL-002. Часовой пояс
- **Правило (OBSERVED):** расписание онлайн-занятий отображается по часовому поясу пользователя.
- **Статус:** INFERRED.

---

## CHAT — Коммуникации

### CHAT-001. Чат с наставником/тьютором
- **Маршрут:** `/chat`.
- **UI:** «Чаты с наставниками» / «Чаты с тьюторами», «Все»/«Непрочитанные», «Список пуст», вложения, «Ответ в течение …», нагрузка (`workload`).
- **API:** `GET /chat/list`, `/chat/single`, `/chat/{id}/message`, `POST /chat/{id}/message`, `GET /chat/workload`.
- **Статус:** INFERRED.

### CHAT-002. Чат поддержки
- **Маршрут:** `/moz`.
- **API:** `GET /chat/moz`.
- **Статус:** INFERRED.

### CHAT-003. Чат комьюнити
- **Описание:** ссылка в программе курса; внешний канал (Telegram/VK) — INFERRED.
- **Статус:** INFERRED.

---

## DUEL — Дуэли

### DUEL-001. Список дуэлей
- **Маршрут:** `/duels`.
- **UI:** «Интерактивные», «Выбери предмет», «Дуэли в ожидании», «История дуэлей», «Победы/Поражения/Ничьи», «У тебя пока нет прошедших дуэлей».
- **API:** `GET /duels`, `GET /duels/{id}`.
- **Статус:** INFERRED.

### DUEL-002. Старт дуэли и матчмейкинг
- **Действие:** «Начать дуэль» → «Мы уже ищем тебе соперника» → уведомление о готовности.
- **API:** `POST /duels/start`.
- **Статус:** INFERRED.

### DUEL-003. Прохождение и результат
- **UI:** «Задание №N», «КИМ №», «Обязательный/Необязательный вопрос», «Завершить», «К списку дуэлей».
- **API:** `POST /duels/{id}/submit`, `GET /duels/{id}/result`.
- **Статус:** INFERRED.

---

## SHOP — Магазин и экономика

### SHOP-001. Магазин соткакоинов
- **Маршрут:** `/coinshop`.
- **UI:** «Магазин», «Мои покупки», «Избранное», «Все», «В наличии N шт.», «Нет в наличии», «Скоро в продаже», «Купить», «Предпросмотр», «Больше не жду :(», «1 шт. в руки», тарифные ограничения (Стандарт/Про/Про+).
- **API:** `GET /coin-product`, `POST /coin-product/{id}/favourite`, `POST /coin-product/buy`.
- **Статус:** INFERRED.

### SHOP-002. Избранное магазина
- **Хранение:** `localStorage` ключ `CoinShopFavoriteProduct`.
- **Статус:** OBSERVED.

### SHOP-003. Тамагочи
- **Маршрут/ассеты:** `/tamagotchi`, `tamgochi-*`, `TamagotchiFloatingBubble-*`.
- **Состояния:** base/sad/weak/tired/dirty/unknown, «Календарь страйка».
- **API:** `POST ${s}/buy-freeze`, `/enable`, `/disable`, `/treat-effect`, `/treat-weakness`.
- **Статус:** INFERRED.

### SHOP-004. Достижения
- **Маршрут:** `/achievements`.
- **UI:** «Достижение», «Задание», «Список пуст», «Пока нет полученных достижений».
- **API:** `GET /achievements`.
- **Статус:** INFERRED.

---

## REF — Реферальная программа

### REF-001. Реферальный кабинет
- **Маршрут:** `/referral`.
- **UI:** «Приглашай друзей в Сотку», «Твой бонусный баланс», «Текущий баланс», «Заработано за всё время», «Доступно для вывода сегодня», «Вывести на карту», «+500 ₽ за оплату пакета обучения другом (вывод через 30 дней)», «10% на месяц обучения…», «Скопировано!».
- **API:** `GET /referral`, `/referral/history`, `/referral/own`, `GET /stats/copyref`, `POST /stats/hit`.
- **Статус:** INFERRED.

### REF-002. Вывод средств
- **API:** `POST /refund/autoWithdrawal`, `/refund/manualWithdrawal`, `GET /refund/checkWithdrawal`, `/refund/history`.
- **Ограничение:** вывод через 30 дней после оплаты приглашённым; номер карты.
- **Статус:** INFERRED.

---

## PROFILE — Профиль и подписки

### PROF-001. Личные данные
- **Маршрут:** `/profile/personal`.
- **Поля:** имя, почта, телефон, VK, дата рождения, ссылки; смена email с кодом.
- **API:** `GET /profile`, `POST /profile`, `POST /profile/change-email/send-code|confirm`.
- **Статус:** INFERRED.

### PROF-002. Мои цели
- **Маршрут:** `/profile/goals`.
- **UI:** «Мои цели», «Настрой персональный план под свои цели и ритм», слайдер балла, дата экзамена (ДД.ММ.ГГГГ), «Сохранить цели».
- **API:** `GET/POST /profile/purposes`.
- **Статус:** INFERRED.

### PROF-003. Подписки
- **Маршрут:** `/profile/subscriptions`.
- **UI:** «Подписки», «Подписка активна», «Подписка закончилась», «Подписка подходит к концу», «Продлить подписку», «Невозможно продлить эту подписку», «Период подписки», «Мои подписки».
- **API:** `GET /profile/storeAccesses`, `/profile/subscriptions`, `/profile/main-courses`, `POST /store/renew`, `/store/updateAccess`.
- **Статус:** INFERRED.

### PROF-004. Реквизиты для выплат
- **API:** `POST /profile/requisitesSaveFromFront`, `GET /profile/voucher/history`.
- **Статус:** INFERRED.

### PROF-005. Обратная связь
- **API:** `POST /feedback`.
- **Статус:** INFERRED.

---

## NOTIF — Уведомления

### NOTIF-001. Центр уведомлений
- **UI:** «Пока нет уведомлений».
- **Типы (OBSERVED):** `chat_message` «Сообщение от наставника», `homework_deadline_today/1d` «Домашняя работа», `homework_checked`, `probnik_deadline_1d`/`probnik_checked` «Пробник», `webinar_starting_soon` «Вебинар», `subscription_ending` «Подписка».
- **API:** `GET /notifications`, WebSocket Centrifugo.
- **Статус:** INFERRED.

---

## PUBLIC — Публичные продукты

| ID | Функция | URL | Статус |
|---|---|---|---|
| PUB-001 | Каталог преподавателей и профили | `/teachers` | CONFIRMED |
| PUB-002 | Предметные лендинги | `/ege/...`, `/oge/...` | CONFIRMED |
| PUB-003 | Учебник | `/uchebnik` | CONFIRMED |
| PUB-004 | Блог с категориями и пагинацией | `/blog` | CONFIRMED |
| PUB-005 | ГДЗ | `/reshebnik` | CONFIRMED |
| PUB-006 | Тарифы | `/about/tariffs` | CONFIRMED |
| PUB-007 | FAQ | `/about/questions` | CONFIRMED |
| PUB-008 | Отзывы | `/about/reviews` | CONFIRMED |
| PUB-009 | Документы/оферты | `/documents` | CONFIRMED |
| PUB-010 | B2B | `/b2b` | CONFIRMED |
| PUB-011 | Для родителей | `/for-parents` | CONFIRMED |
| PUB-012 | Профессии | `/professions` | CONFIRMED |
| PUB-013 | Олимпиады | `/olimpiady` | CONFIRMED |
| PUB-014 | 5–8 классы | `/middleschool` | CONFIRMED |
| PUB-015 | 1–4 классы | `/primaryschool` | CONFIRMED |
| PUB-016 | Нокаут | `/nokaut` | CONFIRMED |
| PUB-017 | Помоги школе | `/helpschool` | CONFIRMED |
| PUB-018 | Амбассадоры | `/ambassadors` | CONFIRMED |
| PUB-019 | Лид-формы (консультация/диагностика) | главная, направления | CONFIRMED (поля), отправка NOT_TESTED |

---

## Сводка

| Категория | Кол-во функций |
|---|---|
| AUTH | 7 |
| REG/ONB | 4 |
| STORE | 7 |
| MAIN/PLAN | 2 |
| COURSE/LESSON | 7 |
| HW | 9 |
| SAMPLE | 3 |
| TASK | 3 |
| STAT | 3 |
| CAL | 2 |
| CHAT | 3 |
| DUEL | 3 |
| SHOP | 4 |
| REF | 2 |
| PROFILE | 5 |
| NOTIF | 1 |
| PUBLIC | 19 |
| **Итого** | **84** |

Из них CONFIRMED — 27, OBSERVED — 8, INFERRED — 45, NOT_TESTED/UNKNOWN — 4.

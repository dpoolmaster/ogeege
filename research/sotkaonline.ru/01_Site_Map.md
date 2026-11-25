# 01. Карта сайта и платформы

Дата исследования: 2026-11-25. Источники: sitemap, публичная навигация, JS-бандл платформы, публичные API.

## 1. Легенда статусов

| Статус | Значение |
|---|---|
| CONFIRMED | Страница/элемент открыты и проверены |
| OBSERVED | Обнаружено в DOM/JS, сценарий не пройден до конца |
| INFERRED | Маршрут/функция восстановлены по коду, без входа |
| NOT_AVAILABLE | Недоступно без аккаунта |
| NOT_TESTED | Обнаружено, но не проверено |

## 2. Публичный сайт (sotkaonline.ru)

Источник: `sitemap.xml` (300 URL), `uchebnik/sitemap/0.xml` (539 URL), меню сайта.

```
sotkaonline.ru
├── /                                   Главная (CONFIRMED)
├── /ege                                ЕГЭ: 10, 11 класс (CONFIRMED)
│   ├── /ege/10class
│   └── /ege/11class
│       └── /ege/11class/{subject}
│           ├── matematika
│           ├── matematika/baza
│           ├── russkii-yazik
│           ├── biologia
│           ├── himia
│           ├── fizika
│           ├── literatura
│           ├── angliskii-yazik
│           ├── obshestvoznanie
│           ├── historia
│           └── informatika
├── /oge                                ОГЭ: 8, 9 класс (CONFIRMED)
│   └── /oge/{9class|8class}/{subject}
│       └── ... + geographya (только 8/9)
├── /olimpiady                          Олимпиады 8–11 (CONFIRMED)
├── /middleschool                       5–8 классы (CONFIRMED)
│   └── /middleschool/{8class|7class|6class|5class}/{subject}
├── /primaryschool                      1–4 классы (CONFIRMED)
│   └── /primaryschool/{4class|3class|2class|1class}/{subject}
├── /about                              О Сотке (CONFIRMED)
│   ├── /about/tariffs                  Тарифы (CONFIRMED)
│   ├── /about/reviews                  Отзывы (CONFIRMED)
│   ├── /about/questions                FAQ (CONFIRMED)
│   ├── /about/contacts                 Контакты (CONFIRMED)
│   ├── /about/media                    Мы в СМИ
│   ├── /about/career                   Карьера
│   ├── /about/vacancies                Вакансии
│   └── /about/recommendations          Рекомендации
├── /teachers                           Каталог преподавателей (CONFIRMED)
│   └── /teachers/{slug}                Профиль преподавателя (CONFIRMED)
├── /sotkastore                         Магазин/конфигуратор (CONFIRMED)
├── /bank-zadaniy                       Бесплатный банк заданий (CONFIRMED)
├── /uchebnik                           Учебник (CONFIRMED)
│   ├── /uchebnik/{subject}
│   └── /uchebnik/{subject}/{topic}
├── /blog                               Блог (CONFIRMED)
│   └── /blog/{slug}
├── /reshebnik                          ГДЗ (CONFIRMED)
├── /documents                          Архив оферт (CONFIRMED)
├── /oferta-page                        Оферта
├── /education                          Сведения об образовательной организации (CONFIRMED)
├── /for-parents                        Для родителей (CONFIRMED)
├── /b2b                                Корпоративное обучение (CONFIRMED)
├── /professions                        Курсы профессий (CONFIRMED)
├── /nokaut                             Интенсив «Нокаут» (CONFIRMED)
├── /helpschool                         «Помоги школе» (CONFIRMED)
├── /ambassadors                        Амбассадоры (CONFIRMED)
├── /hiring, /hr                        HR/найм (в sitemap)
├── /partners/*                         Партнёрские лендинги (132 URL в sitemap)
├── /info, /unsub, /oprosnik-ddx        Служебные страницы
├── /popd                               Политика ПДн
└── /404                                Страница не найдена
```

### 2.1 Глобальная навигация (CONFIRMED)

Верхнее меню: `ЕГЭ`, `ОГЭ`, `Олимпиады`, `5-8 классы`, `1-4 классы`, `Другие направления` (Блог, Банк заданий, ГДЗ, Профориентация, Учебник), `О нас` (О Сотке, Отзывы, Преподаватели, Контакты, Мы в СМИ, Карьера, Вопросы и ответы), `Тарифы`, `Купить`, `Войти`.

Футер: О нас, Наши продукты, Подготовка, Для пользователей платформы, контакты, юридические данные, оферты, лицензия.

### 2.2 Ключевые страницы и их назначение

| URL | Назначение | Роли | Статус |
|---|---|---|---|
| `/` | Продажа, УТП, тарифы, отзывы, FAQ, лид-форма | Guest | CONFIRMED |
| `/ege`, `/oge` | Выбор класса и предмета | Guest | CONFIRMED |
| `/ege/11class/matematika` | Предметный лендинг с программой, преподавателями, FAQ, ценами | Guest | CONFIRMED |
| `/about/tariffs` | Сравнение 5 тарифов, помесячно/год | Guest | CONFIRMED |
| `/sotkastore` | Выбор класса → предметов → тарифа → оформление | Guest | CONFIRMED (UI), оплата NOT_TESTED |
| `/bank-zadaniy` | Бесплатные задания: предмет → тема → вариант → ответ | Guest | CONFIRMED |
| `/uchebnik` | Каталог теории по предметам/темам | Guest | CONFIRMED |
| `/blog` | Статьи, категории, пагинация | Guest | CONFIRMED |
| `/reshebnik` | ГДЗ по учебникам | Guest | CONFIRMED |
| `/teachers` | Каталог преподавателей | Guest | CONFIRMED |
| `/about/questions` | FAQ (аккордеоны) | Guest | CONFIRMED (вопросы), ответы частично |
| `/documents` | Редакции оферт | Guest | CONFIRMED |
| `/education` | Сведения об образовательной организации | Guest | CONFIRMED |

### 2.3 Публичные API сайта (OBSERVED)

| Метод | Endpoint | Назначение |
|---|---|---|
| GET | `admin.sotkaonline.ru/api/v1/settings/auth` | Конфиг авторизации (`{"type":"new"}`) |
| GET | `/api/v1/store/sotka-grades?special=0` | Доступные классы магазина (9,10,11) |
| GET | `/api/v1/store/sotka-courses?grade=&year_type=` | Курсы/предметы для магазина |
| GET | `/api/v1/store/sotka-products/?grade=&special=0` | Продукты, тарифы, цены, периоды доступа |
| GET | `/api/v1/public-task-bank/courses?grade=` | Курсы публичного банка заданий |
| GET | `/api/v1/public-task-bank/homeworks/{id}` | Вариант и задания банка |
| POST | `auth.sotkaonline.ru/api/v2/send-verify-code` | Отправка кода входа |

## 3. Платформа (platform.sotkaonline.ru)

### 3.1 Публичные/полупубличные маршруты

| Маршрут | Компонент | Доступ | Статус |
|---|---|---|---|
| `/auth` | AuthPage | Guest | CONFIRMED |
| `/auth-middle` | MiddleAuthPage | Guest | INFERRED |
| `/register` | RegisterPage | Guest/после покупки | INFERRED |
| `/` | EntryPage (редирект) | Guest | OBSERVED |
| `/statistics/:id` | PublicStatisticsPage | Публичный отчёт | CONFIRMED (ошибка на неверном id) |
| `*` | NotFoundPage | Guest | OBSERVED |

**Факт:** любой защищённый маршрут (`/mainpage`, `/courses`, `/homework`, `/calendar`, `/duels`, `/coinshop`, `/profile`, `/chat`, `/library`, `/tasksStorage`, `/samples`, `/daily-plan`, `/notifications`, `/textbook`, `/trial`, `/register`) без сессии редиректит на `/auth`. Проверено для 24 маршрутов (**CONFIRMED**).

### 3.2 Карта ЛК (INFERRED из роутера и навигации)

```
/mainpage                         Главная: план, эфиры, прогресс, реферал
/daily-plan                       План на сегодня: теория, практика, пропущенное
/courses                          Мои предметы
└── /courses/:courseId            Программа курса (блоки, темы, уроки, зачёты)
    └── /courses/:courseId/:lessonId   Урок (видео/статья/тест/практика)
/homework                         Домашки: к выполнению/выполнено
└── /homework/:hwId               Выполнение ДЗ
    └── /homework/:hwId/result    Результат
/tagHomework/:hwId                Тематическая домашка (теги)
/selfcheck                        Самопроверка
└── /selfcheck/:hwId              Самопроверка задания
/samples                          Пробники/контрольные
/tasksStorage                     Банк заданий (в ЛК)
/library                          Библиотека материалов
/statistics                       Аналитика
/statistics/:id                   Публичный отчёт
/calendar                         Расписание (календарь)
/timetable                        Расписание (список/сетка)
/chat                             Чаты с наставниками/тьюторами
/moz                              Чат поддержки
/duels                            Интерактивные дуэли
└── /duels/:courseId              Дуэль
    └── /duels/:courseId/result   Результат
/coinshop                         Магазин за соткакоины
/achievements                     Достижения
/referral                         Реферальная программа
/profile                          Профиль
├── /profile/personal             Личные данные
├── /profile/subscriptions        Подписки
└── /profile/goals                Мои цели
/textbook                         Учебник в ЛК
/notifications                    Уведомления
/tamagotchi                       Тамагочи (питомец)
/pay-agreement*                   Оферты (15/61/90/123/150 дней, ВУЗ, СПО, ССУЗ)
/payback-agreement                Оферта возврата
```

### 3.3 Навигация ЛК (CONFIRMED по конфигу бандла)

```
Главная                /mainpage
План на сегодня        /daily-plan
Обучение
├── Мои предметы       /courses
├── Статистика         /statistics
├── Домашки            /homework
├── Чат с наставником  /chat        (или «Чат с тьютором» при hasTutorProduct)
├── Чат поддержки      /moz         (только isHighSchool)
├── Пробники           /samples     (или «Контрольные» для 8 класса/младших)
├── Банк заданий       /tasksStorage
└── Библиотека         /library
Расписание             /calendar (или /timetable; для trial — /courses)
Ещё
└── Реферальная программа /referral
```

Дополнительно доступны (вне основного меню): `/duels`, `/coinshop`, `/achievements`, `/profile`, `/notifications`, `/textbook`.

### 3.4 Сегментация навигации (OBSERVED)

Флаги, влияющие на видимость/блокировку пунктов:

| Флаг | Влияние |
|---|---|
| `isTrialUser` | Блокирует Главную, Домашки, Пробники, Библиотеку; показывает заглушки |
| `isTutor` | Скрывает «Обучение», «Расписание» |
| `isHighSchool` | Показывает «Чат поддержки» (`/moz`) |
| `isMiddleLow`, `isZeroYearType`, `grade_8` | Меняет «Пробники» на «Контрольные» |
| `hasTutorProduct` | Меняет «Чат с наставником» на «Чат с тьютором» |
| `grade <= 8` | Упрощённое меню, скрывает часть пунктов |
| `isProfessionsMode` | Оставляет только «Мои предметы» |

## 4. Роли и доступ (сводно)

| Роль | Доступ | Статус |
|---|---|---|
| Guest | Публичный сайт, банк заданий, учебник, блог, ГДЗ, тарифы, магазин, форма входа | CONFIRMED |
| Student (paid) | Полный ЛК | INFERRED |
| Trial student | Ограниченный ЛК (план знакомства, часть разделов заблокирована) | INFERRED |
| Parent | Просмотр статистики ребёнка (`/parent/statistic/*`) | INFERRED |
| Tutor | Отдельный режим (`is_tutor`, `/tutors/register`) | INFERRED |
| Teacher | Регистрация/вход преподавателя (`/teacher/*`) | INFERRED |
| Mentor/Curator | Проверка ДЗ, чат, апелляции | INFERRED |
| Support | Чат поддержки `/moz` | INFERRED |
| Admin | Внутренняя админка (не исследовалась) | NOT_AVAILABLE |

Подробнее: `07_Roles_and_Permissions.md`.

## 5. Что не удалось картировать

- Внутренняя админка (`admin.sotkaonline.ru`) — только публичные API.
- Точные экраны ЛК (визуально) — нет аккаунта.
- Платёжные шлюзы и статусы платежей — оплата не выполнялась.
- Родительский кабинет и кабинет преподавателя — нет доступа.
- Партнёрские страницы (132 URL) — не исследовались поштучно.
- Блог (62 страницы пагинации) — исследован только первый экран.

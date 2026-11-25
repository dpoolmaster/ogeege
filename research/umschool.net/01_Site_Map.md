# 01. Карта сайта

Дата: 25.11.2026
Статусы: `CONFIRMED` — проверено в браузере/API; `OBSERVED` — ссылка/маршрут обнаружен; `NOT_AVAILABLE` — недоступно аккаунту; `NOT_TESTED` — есть, но не проверялось.

---

## 1. Дерево публичного сайта `umschool.net`

```
umschool.net
├── /                                   Главная (лендинг)                         CONFIRMED
├── /courses/                           Каталог курсов                             CONFIRMED
│   ├── /courses/free/                  Бесплатные курсы                           CONFIRMED
│   └── /courses/new-bundle/            Конфигуратор комплекта (публичный)         OBSERVED
├── /ege/                               Хаб ЕГЭ                                    CONFIRMED
│   ├── /ege/11-class/                  ЕГЭ 11 класс                               OBSERVED
│   ├── /ege/10-class/                  ЕГЭ 10 класс                               OBSERVED
│   ├── /ege/{subject}/                 Предметный лендинг ЕГЭ                     OBSERVED
│   ├── /ege/{class}/{subject}/         Класс + предмет                            OBSERVED
│   └── /ege/{class}/{subject}/teacher--{id}/  Курсы преподавателя                  OBSERVED
├── /oge/                               Хаб ОГЭ                                    CONFIRMED
│   ├── /oge/{subject}/                 Предметный лендинг ОГЭ                     OBSERVED
│   └── /oge/9-class/{subject}/         ОГЭ 9 класс + предмет                      OBSERVED
├── /srednyaya-shkola/                  Средняя школа 5–8 классы                   CONFIRMED
│   ├── /srednyaya-shkola/7-class/      Курсы 7 класса                             OBSERVED
│   ├── /srednyaya-shkola/8-class/      Курсы 8 класса                             OBSERVED
│   └── /srednyaya-shkola/{class}/{subject}/  Предмет 5–8 класса                    OBSERVED
├── /olimpiady/                         Олимпиадная подготовка                     CONFIRMED
│   ├── /olimpiady/{subject}/           Предмет олимпиад                           OBSERVED
│   └── /olimpiady/11-class/            Олимпиады 11 класс                         OBSERVED
├── /tutor/                             Каталог преподавателей                     CONFIRMED
│   └── /tutor/{slug}/                  Профиль преподавателя                      CONFIRMED
├── /parents/                           Лендинг для родителей                      CONFIRMED
├── /reviews/                           Отзывы + фильтры + пагинация               CONFIRMED
├── /faq/                               FAQ (категории-аккордеоны)                 CONFIRMED
├── /contacts/                          Контакты, реквизиты                        CONFIRMED
├── /sposoby-oplaty/                    Способы оплаты                             CONFIRMED
├── /coupons/                           Промокоды (хаб)                            OBSERVED
├── /coupons/free/                      Бесплатные промокоды/курсы                 CONFIRMED
├── /free/                              Бесплатные инструменты (редирект)          CONFIRMED
│   ├── /free/bank-zadaniy/             Банк заданий ЕГЭ/ОГЭ                       CONFIRMED
│   ├── /free/bank-zadaniy/{exam}/{subject}/{section}/  Раздел банка              OBSERVED
│   ├── /free/probniki/                 Пробные варианты                           CONFIRMED
│   ├── /free/probniki/{exam}/{subject}/  Пробники по предмету                     CONFIRMED
│   ├── /free/emulyator-kege/           Эмулятор КЕГЭ                              CONFIRMED (HTTP 500)
│   └── /free/interaktivnye-trenazhery/ Интерактивные тренажёры                    CONFIRMED
├── /about/                             О школе, история, миссия                   CONFIRMED
├── /mobileapp/                         Мобильное приложение                       OBSERVED
├── /sitemap/                           Карта сайта                                OBSERVED
├── /documents/agreement/               Согласие на обработку ПД                   OBSERVED
├── /documents/privacy-policy/          Политика обработки данных                  OBSERVED
├── /documents/educational-license/     Образовательная лицензия                   OBSERVED
├── /documents/oferta/                  Договор оферты                             OBSERVED
├── /reestr/documents/{hash}/           Реестровые документы ПО                    OBSERVED
├── /cart/                              Публичная корзина                          OBSERVED
├── landing.umschool.net/referral       Реферальная программа (лендинг)            OBSERVED
├── landing.umschool.net/collaboration  Сотрудничество                             OBSERVED
├── landing.umschool.net/partners/      Партнёры                                   OBSERVED
├── landing.umschool.net/pwa/           iOS-приложение                             OBSERVED
├── umschool.net/library/               Учебник/библиотека                         OBSERVED
├── umschool.net/journal/               Журнал                                     OBSERVED
├── umschool.one                        Репетиторы (отдельный лендинг)             CONFIRMED
├── promo.umschool.net                  Акционные лендинги (Tilda)                 OBSERVED
└── cpa.umschool.net                    Партнёрская программа                      OBSERVED
```

Публичный `robots.txt` закрывает от индексации `/accounts/`, `/lessons/`, `/homework/`, `/profile/`, `/signup/`, `/live/`, `/mastergroup/`, `/predbannik/`, `/api/`, `/private/`, `/froala/`. Sitemap содержит 673 URL, из них 357 — `/free/**`, 80 — `/course/**`, 66 — `/ege/**`, 61 — `/tutor/**`, 40 — `/oge/**`, 34 — `/srednyaya-shkola/**`, 20 — `/olimpiady/**`.

---

## 2. Дерево авторизации `id.umschool.net`

| Маршрут | Назначение | Статус |
|---|---|---|
| `/login` | Вход/регистрация по телефону + код | CONFIRMED |
| `/login/email` | Вход по e-mail + паролю | CONFIRMED |
| `/recover-password` | Восстановление пароля по e-mail | CONFIRMED |
| `/change-password` | Смена пароля (из профиля) | CONFIRMED |
| `/fast-login` | Быстрый вход (маршрут в бандле) | NOT_TESTED |
| `/platform-choose` | Выбор платформы (RU/ONE/KZ/HEAD) | NOT_TESTED |
| `/conformation-consent` | Подтверждение согласия | NOT_TESTED |
| `/confirm-advertising` | Согласие на рекламные рассылки | NOT_TESTED |
| `/callback/email-change` | Callback смены e-mail | NOT_TESTED |
| `/internal-error` | Внутренняя ошибка | NOT_TESTED |
| `/not-found` | 404 | NOT_TESTED |

OAuth-вход в LMS: `https://id.umschool.net/?client_id=...&client_redirect=https%3A%2F%2Fnd.umschool.net%2Foauth-callback`, затем `nd.umschool.net/oauth-callback?token=...`.

---

## 3. Дерево LMS `nd.umschool.net`

Роли в маршрутах из фронтенд-бандла: `Student` (основная), `ChatModerator` (урок), `Guest` (спец. страницы).

```
nd.umschool.net
├── /home                                  Дашборд ученика                          CONFIRMED
├── /schedule/:calendarParams*/            Расписание (неделя)                      CONFIRMED
├── /lessons                               Мои курсы/предметы                       CONFIRMED
│   ├── /lessons/:productTypeId            Занятия курса/предмета                    CONFIRMED
│   └── /lessons/:productTypeId/modules/:moduleId  Модуль курса                      OBSERVED
├── /lesson/:lessonId                      Плеер занятия                             CONFIRMED
├── /lesson/access/:lessonHash             Доступ по hash (шаринг)                   OBSERVED
├── /homework                              Редирект/служебный                        OBSERVED
│   ├── /homework/list/:params*            Список ДЗ + фильтры                       CONFIRMED
│   ├── /homework/:homeworkId              Выполнение ДЗ                             CONFIRMED
│   ├── /homework/:homeworkId/work-on-bugs Работа над ошибками                       CONFIRMED
│   ├── /homework/:homeworkId/completed    Результаты ДЗ                             CONFIRMED
│   ├── /homework/:submissionId/print      Версия для печати                         CONFIRMED
│   ├── /homework/generated/:submissionId  Сгенерированное ДЗ                        OBSERVED
│   └── /homework/generated/:submissionId/result  Результат сгенерированного ДЗ      OBSERVED
├── /progress                              Тренажёр/аналитика                        CONFIRMED
├── /progress/class-type/selection         Онбординг: класс и предметы               CONFIRMED
├── /training/:trainingType                Сессия тренажёра (cards/...)              CONFIRMED
├── /training/settings                     Настройки тренировки                      CONFIRMED
├── /training/result                       Результат тренировки                      CONFIRMED
├── /level-test/:testId                    Тест определения уровня                   CONFIRMED (ошибка теста 1)
├── /catalog                               Хаб магазина                              CONFIRMED
├── /catalog/:sysCourseTypeCode/:productTypeId  Каталог/карточка продукта            CONFIRMED
├── /catalog/:sysCourseTypeCode/bundles/configure  Конфигуратор комплекта              CONFIRMED
├── /cart                                  Корзина                                    CONFIRMED
├── /cart-owned-by-another                 Корзина по ссылке другого пользователя     OBSERVED
├── /subscriptions                         Подписки                                   CONFIRMED
│   ├── /subscriptions/:id/renewal         Продление                                  NOT_AVAILABLE
│   └── /subscriptions/:id/upgrade         Улучшение тарифа                           NOT_AVAILABLE
├── /loyalty                               Достижения/уровни                          CONFIRMED
├── /loyalty/about                         О программе достижений                     CONFIRMED
├── /referral/invite                       Пригласи друга                             CONFIRMED
├── /referral/history                      История бонусов                            CONFIRMED
├── /chat                                  Онлайн-куратор                             CONFIRMED (empty)
├── /documents                             Документы                                  CONFIRMED
├── /profile                               Профиль                                    CONFIRMED
├── /mentor                                Внешний переход в VK                       CONFIRMED (redirect vk.ru)
├── /results                               Редирект на /home                          CONFIRMED
├── /onboarding                            Онбординг/нет активных курсов              CONFIRMED
├── /onboarding/complete                   Завершение онбординга                      CONFIRMED
├── /switch, /switch/na                    Переключение платформы                     CONFIRMED (redirect /home)
├── /payment/success/:transactionUuid      Успешная оплата                            OBSERVED
├── /coupon-callback                       Callback промокода                         OBSERVED
├── /merch-callback                        Callback мерча                             OBSERVED
├── /dec-callback                          Callback DEC                               OBSERVED
├── /dnevnik-callback                      Callback Дневник.ру                        OBSERVED
├── /login                                 Логин (SPA-заглушка/редирект)              CONFIRMED
├── /forbidden                            403                                        OBSERVED
├── /bad-request                          400                                        OBSERVED
├── /unexpected                            Неожиданная ошибка                         OBSERVED
└── /not-found                            404                                        OBSERVED
```

---

## 4. Основная навигация LMS

### 4.1. Левая боковая панель (desktop)

Порядок из DOM:

1. Магазин → `/catalog`
2. Главная → `/home`
3. Расписание → `/schedule/`
4. Занятия → `/lessons`
5. Домашние работы → `/homework/list`
6. Тренажёр → `/progress`
7. Онлайн-куратор → `/chat`
8. Достижения → `/loyalty`
9. Пригласи друга → `/referral/invite`
10. Подписки → `/subscriptions`

Внизу панели: счётчики медалей, умкоинов, серии (strike). Клик по счётчику открывает поповер с пояснением и ссылкой.

### 4.2. Верхняя панель

- Логотип/навигация на мобильных.
- Корзина (`/cart`) с бейджем количества.
- Аватар «ДН» → меню профиля (в исследовании не раскрылось из-за overlay; маршрут профиля доступен напрямую `/profile`).
- Промо-баннер «Забрать декабрь со скидкой» / «ВЕБЫ ИДУТ БЕЗ ТЕБЯ» (Popmechanic).
- Кнопки «1 / 3 / 0» — медали / умкоины / серия.

### 4.3. Мобильная навигация

- Бургер (`icon-burger-default`) открывает боковое меню.
- Скрыта боковая панель, контент в одну колонку; на `/schedule/` календарь компактный.
- На `/homework/list` фильтры открываются кнопкой «Фильтры».

---

## 5. Публичная навигация (header/footer)

Header: `Курсы`, `ЕГЭ`, `ОГЭ`, `Родителям`, `Бесплатно`, `Прочее`, `О нас`, `Выбрать предметы`, номер телефона, `Войти`, `Корзина`.

Footer: `Умскул`, `Обучение` (ЕГЭ, ОГЭ, Каталог, Пробные варианты), `О компании` (О нас, Преподаватели, Контакты, FAQ, Наши партнёры, Сведения об образовательной организации, Карта сайта, Сотрудничество, Способы оплаты, Все отзывы), `Образовательная платформа «Умскул»` (реестровые документы), `Наши продукты` (Журнал, мобильное приложение, iOS/Android, Учебник, Стать партнёром, Приглашай друга), контакты, соцсети, лицензия, оферта, политика.

---

## 6. Статусы исследования страниц

| Раздел | Статус | Комментарий |
|---|---|---|
| Главная LMS | Изучена | Все блоки, API |
| Расписание | Изучена частично | Нет событий в демо-курсе; календарь/URL/пустое состояние |
| Занятия | Изучена | 1 курс, 2 занятия, фильтры |
| Плеер занятия | Изучена | Видео Kinescope, таймкоды, материалы (0), ДЗ |
| Домашние работы | Изучена | Полный цикл демо-анкеты, результаты, работа над ошибками, печать |
| Тренажёр | Изучена | Настройки, сессия, оценки, результат, 42 835 карточек |
| Аналитика | Изучена частично | Темы, пробники (empty), вкладки «Номера КИМ», «График» |
| Каталог | Изучена | Хаб, списки, карточка, тарифы, комплект |
| Корзина | Изучена | Пустое/заполненное, подарок, промокод, ссылка |
| Checkout | Изучена без оплаты | Форма, способы оплаты, скидки |
| Подписки | Empty state | Нет активных подписок |
| Достижения | Изучена | Уровни, 8 ачивок, купоны, история |
| Реферал | Изучена частично | Нет принятия условий/ссылки |
| Чат | Empty state | Нет диалогов |
| Профиль | Изучена | Поля, модалки, пароль, VK |
| Документы | Empty state | Нужна дата рождения |
| Публичный сайт | Изучен выборочно | Ключевые лендинги и инструменты |
| Админка/роли | Недоступно | Только перечень ролей из бандла |
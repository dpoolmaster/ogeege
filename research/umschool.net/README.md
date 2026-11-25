# Research: umschool.net

Исследование платформы подготовки к ОГЭ/ЕГЭ **UMSCHOOL** (25.11.2026).
Авторизованный тестовый аккаунт ученика; публичный сайт; read-only API; frontend-бандлы.
Реальные платежи и необратимые действия не выполнялись.

## Документы

| Файл | Содержание |
|---|---|
| [00_Overview.md](00_Overview.md) | Обзор продукта, домены, роли, ценность, ограничения |
| [01_Site_Map.md](01_Site_Map.md) | Публичная и LMS-карта, маршруты, навигация |
| [02_Features_Catalog.md](02_Features_Catalog.md) | 176 функций в 17 категориях |
| [03_User_Flows.md](03_User_Flows.md) | 21 end-to-end сценарий + Mermaid |
| [04_Business_Logic.md](04_Business_Logic.md) | 78 бизнес-правил |
| [05_Data_Model.md](05_Data_Model.md) | Сущности, поля, enum, ER-диаграмма |
| [06_UI_Components.md](06_UI_Components.md) | Layout, дизайн-система, компоненты, адаптивность |
| [07_Roles_and_Permissions.md](07_Roles_and_Permissions.md) | Роли, матрица доступа, ограничения |
| [08_Notifications_and_Integrations.md](08_Notifications_and_Integrations.md) | Уведомления и внешние интеграции |
| [09_Implementation_Specification.md](09_Implementation_Specification.md) | A/B/C: observed / recommended / unknown |
| [10_MVP_and_Roadmap.md](10_MVP_and_Roadmap.md) | MVP, Phase 2/3, roadmap, риски |
| [11_Unknowns_and_Gap_Analysis.md](11_Unknowns_and_Gap_Analysis.md) | Пробелы, вопросы, предположения, backlog |
| [12_Research_Log.md](12_Research_Log.md) | Подробный журнал исследования |
| [13_Screenshots/](13_Screenshots/README.md) | 85 скриншотов с Feature ID |
| [14_Requirements_Backlog.md](14_Requirements_Backlog.md) | 121 требование с критериями приёмки |
| [15_Test_Cases.md](15_Test_Cases.md) | 97 тест-кейсов |

## Сырые материалы

`_raw/` — API-дампы, HTML, тексты, скрипты сбора и оригинальные скриншоты. Персональные данные и токены санитизированы.

## Ключевые выводы

- Продукт: LMS + подписочная коммерция + каталог + геймификация + чат + публичные бесплатные инструменты.
- Основной стек (наблюдаемый): Vue 3 SPA (`nd.umschool.net`), SSO (`id.umschool.net`), REST API (`old.umschool.net`), checkout (`pay.umschool.net`), чат (`chat-api` + Centrifugo), CDN, Kinescope.
- 176 функций, 21 сценарий, 78 правил, 61 уровень, 8 достижений, 76 158 публичных заданий.
- Ограничения: один ученический аккаунт без покупок/подписок/чата/админки.
- Покрытие функций: ~83% (128 CONFIRMED, 31 OBSERVED, 9 INFERRED, 5 NOT_AVAILABLE, 3 NOT_TESTED).
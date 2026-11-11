# На чём основана концепция

Дата изучения: 11 ноября 2026 года. Проверены локальные исходники двух проектов и материалы, найденные через Exa MCP. Приложения в production и их внешние API не запускались.

## Локальные проекты

| Источник | Что учитываем |
| --- | --- |
| [SruDoc: README-ru.md](../../srunal_docgen/README-ru.md) | Назначение, заказы, текущий Kaspi-сценарий, генерация, доставка и доработка |
| [SruDoc: App.tsx](../../srunal_docgen/miniapp/src/App.tsx) | Реальные экраны клиента и оператора, обработка параметров запуска |
| [SruDoc: server.go](../../srunal_docgen/backend/internal/api/server.go) | Текущие API, клиентский вход и отдельные права оператора |
| [SruDoc: initdata.go](../../srunal_docgen/backend/internal/auth/initdata.go) | Проверка Telegram-личности по `initData` |
| [SruDay: README.md](../../sruday/README.md) | Назначение и существующие сценарии календаря |
| [SruDay: App.tsx](../../sruday/web/src/App.tsx) | Реальные маршруты, включая экран сообщества |
| [SruDay: CommunityScreen.tsx](../../sruday/web/src/pages/Community/CommunityScreen.tsx) | Создание групп, вступление по коду, участники и общее свободное время |
| [SruDay: server.go](../../sruday/backend/internal/httpapi/server.go) | Маршруты авторизации, событий, друзей, групп, импорта и подключений |
| [SruDay: worker.go](../../sruday/backend/internal/worker/worker.go) | `connector_sync`, `digest` и `retention` ещё не реализованы |
| [SruDay: bot.go](../../sruday/backend/internal/bot/bot.go) | Уже есть обработка `/today`, хотя README описывает бота как не начатого |

Ссылки на соседние проекты рассчитаны на текущую структуру папок. После публикации этого репозитория отдельно от соседних проектов они могут быть недоступны.

Документация соседних проектов местами отстаёт от исходников. Поэтому наличие маршрута или экрана отмечено по коду, а готовность интеграции или production-поведение не предполагается. Например, маршрут синхронизации подключения не означает, что коннектор уже умеет получать расписание.

По манифестам оба фронтенда уже используют React 19. README проекта DocGen ещё упоминает React 18. Пользовательское название этого приложения в SruNal теперь SruDoc; локальный проект `srunal_docgen` сохраняется. Для новых данных SruNal отдельно выбрана PostgreSQL 18, старые базы не мигрируются автоматически.

## Поиск через Exa

Поиск охватил три направления: функции учебных планировщиков, ограничения Telegram Mini Apps и устройство сообществ с лентой и чатами. Запрошено 15 поисковых результатов, затем отдельно прочитаны наиболее полезные первичные страницы. Повторяющиеся страницы учитывались как один источник при выводах.

| Проверенный источник | Что подтверждает | Как используем |
| --- | --- | --- |
| [Telegram: запуск Mini Apps](https://core.telegram.org/bots/webapps#direct-link-mini-apps) | Прямые ссылки, `startapp`, отсутствие доступа к переписке чата из такого запуска | Первое подключение самостоятельных приложений; собственное хранение чатов соцсети |
| [Telegram: проверка данных](https://core.telegram.org/bots/webapps#validating-data-received-via-the-mini-app) | Серверная проверка подписи и `auth_date` | Граница доверия и вход между приложениями |
| [Telegram: проверка третьей стороной](https://core.telegram.org/bots/webapps#validating-data-for-third-party-use) | Проверка без передачи секретного токена бота | Возможный вариант будущей авторизации интеграций |
| [Telegram: цифровые товары и услуги](https://core.telegram.org/bots/payments-stars) | Требование Telegram Stars для цифровых продаж внутри Telegram | Отдельная задача адаптации платного SruDoc |
| [MyStudyLife: возможности](https://mystudylife.com/tour/) | Расписание, задания и экзамены как связанные учебные сценарии | Контекст для связи SruDay с другими приложениями; не доказательство спроса на SruNal |
| [Circle: обсуждения](https://circle.so/discussions) | Раздельные форматы постов и общения в реальном времени | Лента и чаты как социальная основа самой SruNal |
| [Circle: навигация](https://help.circle.so/p/basics/getting-started/navigating-circle-on-desktop) | Сообщества, личные и групповые диалоги, профили, события | Ориентир по составу социальных разделов SruNal |

Telegram является первичным источником технических ограничений. MyStudyLife и Circle описывают собственные продукты, поэтому их страницы подходят для изучения сценариев, но не подтверждают рыночные преимущества или востребованность наших идей.

После обсуждения пользователь выбрал SruPulse, SruNotes и SruSplit дополнительно к SruDoc и SruDay. SruNal определена как сама соцсеть с лентой, чатами и каталогом этих приложений. SruNotes ограничена простыми заметками с возможностью делиться. Это продуктовые требования пользователя; поиск служит источником технических ограничений и ориентиров.

## Поиск навыка планирования

По запросу пользователя применён локальный навык `find-skills`: сначала просмотрен [каталог skills.sh](https://skills.sh/), затем изучены [to-prd](https://skills.sh/mattpocock/skills/to-prd) и [исходный репозиторий mattpocock/skills](https://github.com/mattpocock/skills).

`to-prd` подходит как ориентир для превращения разговора и состояния репозитория в требования. Взята структура: проблема пользователя, желаемое поведение, границы, проверка результата. Это дополнительный источник методики; новый навык не устанавливался и не запускал публикацию в трекер. Местом результата по заданию пользователя остаётся `docs/`.

Также использованы уже доступные навыки Ponytail и Telegram Mini App: минимальный первый объём, сохранение текущих сценариев, серверная авторизация и особенности мобильного клиента Telegram.

## Уточнение архитектуры и безопасности

Для архитектуры дополнительно изучены официальные документы через Exa и выполнен целевой поиск Bot API на пять результатов. Использованы навыки PostgreSQL, Go Project Layout и Mobile App UI Design. Пользователь определил уведомления в чате бота и UI/UX существующего DocGen; эти решения имеют приоритет над общими рекомендациями навыков.

| Источник | Как используется |
| --- | --- |
| [PostgreSQL: поддерживаемые версии](https://www.postgresql.org/support/versioning/) | Проверка поддержки выбранной PostgreSQL 18 |
| [PostgreSQL: ограничения](https://www.postgresql.org/docs/18/ddl-constraints.html) | Связи, уникальность и границы проверок целостности |
| [PostgreSQL: SELECT](https://www.postgresql.org/docs/18/sql-select.html) | Захват заданий через SKIP LOCKED |
| [PostgreSQL: backup](https://www.postgresql.org/docs/18/backup.html) | Резервное копирование и проверка восстановления |
| [Go: net/http](https://pkg.go.dev/net/http) | HTTP API и webhook на стандартной библиотеке |
| [Go: crypto/cipher](https://pkg.go.dev/crypto/cipher#NewGCMWithRandomNonce) | Стандартное authenticated encryption со случайным nonce |
| [Vite: static deploy](https://vite.dev/guide/static-deploy.html) | Production-сборка SPA как статики |
| [TanStack Query](https://tanstack.com/query/latest/docs/framework/react/overview) | Серверное состояние фронтенда |
| [Telegram: webhook](https://core.telegram.org/bots/api/methods/setwebhook.md) | Secret header и единый способ приёма updates |
| [Telegram: retry_after](https://core.telegram.org/bots/api/types/responseparameters.md) | Повтор доставки после ограничения частоты |
| [OWASP: cryptographic storage](https://cheatsheetseries.owasp.org/cheatsheets/Cryptographic_Storage_Cheat_Sheet.html) | Разделение ключей, authenticated encryption и безопасная случайность |
| [OWASP: authorization](https://cheatsheetseries.owasp.org/cheatsheets/Authorization_Cheat_Sheet.html) | Проверка прав при каждом обращении к объекту |
| [OWASP: sessions](https://cheatsheetseries.owasp.org/cheatsheets/Session_Management_Cheat_Sheet.html) | Защита и отзыв сессионных секретов |

Основа UI подтверждена по [CSS-токенам](../../srunal_docgen/miniapp/src/index.css), [компонентам](../../srunal_docgen/miniapp/src/components/ui.tsx) и [темам](../../srunal_docgen/miniapp/src/app/theme.ts). В SruDay изучен существующий [vault](../../sruday/backend/internal/vault/vault.go) как пример серверного шифрования и ротации. Новая модель безопасности требует также аутентифицированного контекста полей и отдельного хранения ключей; существующие сервисы не объявляются проверенными этой документацией.

## Что остаётся гипотезой

Востребованность сценариев SruNal, частота действий между приложениями и стоимость поддержки ещё не измерены. Социальная основа и состав приложений уже выбраны пользователем. Закрытый пилот проверяет удобство и детали реализации этого состава.

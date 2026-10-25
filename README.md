# Sparkly

- `backend/` — Control Plane на Go.
- `frontend/` — SvelteKit.
- `agent/` — Rust.
- `ml/` — Python.
- `analytics/` — аналитика на Go.
- `envoy/` — gateway: маршрутизация сервисов и JWT-аутентификация из cookie.
- `docs/` — архитектура и решения.

Protobuf-контракты хранятся в каталогах `proto/` соответствующих сервисов. Задачи проектов запускаются через Nx; JavaScript-зависимости управляются pnpm.

## Разработка

Требуются Node.js 22.18+, pnpm 11.7.0 и родные инструменты проектов: Go, Rust/Cargo, Python.

```sh
pnpm install
pnpm dev                       # frontend
pnpm check                     # проверки проектов и Protobuf-контрактов
pnpm build                     # frontend, backend, analytics и agent
pnpm nx run backend:dev        # запуск отдельного проекта
pnpm nx run ml:dev
pnpm graph
```

Nx использует `project.json` в каждом проекте. Кеш Nx включён для сборки Go и frontend.

## Docker Compose

Для каждого приложения, кроме agent, добавлен Dockerfile. Compose также запускает PostgreSQL 18 и ClickHouse 26.8 из официальных образов.

| Сервис / профиль       | Порт хоста по умолчанию    |
| ---------------------- | -------------------------- |
| frontend               | 3000                       |
| envoy                  | 10000                      |
| backend                | 8080                       |
| analytics              | 8081                       |
| ml                     | 8082                       |
| postgres (без профиля) | 5432                       |
| clickhouse             | 8123 (HTTP), 9000 (native) |

```sh
# Все профили
docker compose --profile '*' up --build -d

# Только выбранные сервисы (backend)
docker compose --profile backend up --build -d
```

Gateway доступен на `http://localhost:10000` (порт задаётся через `ENVOY_PORT`).
Для frontend через gateway задайте `FRONTEND_ORIGIN=http://localhost:10000`.
Frontend и RPC регистрации/входа публичны, остальные маршруты сервисов требуют
cookie `access_token`.

Во всех контейнерах `extra_hosts` направляет имена `frontend`, `backend`, `analytics`, `ml`, `postgres`, `clickhouse` и `host.docker.internal` на `host-gateway`. Поэтому запрос `postgres:5432` идёт через опубликованный порт хоста. Использовать `127.0.0.1` в этих записях нельзя: это loopback самого контейнера. Снаружи Docker сервисы доступны через `localhost:<порт>`; записи `extra_hosts` не меняют DNS хоста или браузера.

Чтобы заменить backend локальным процессом:

```sh
docker compose stop backend
docker compose --profile frontend up --build -d
POSTGRES_HOST=localhost POSTGRES_PORT=5432 POSTGRES_DB=sparkly \
  POSTGRES_USER=sparkly POSTGRES_PASSWORD=sparkly \
  JWKS_MASTER_KEY=sparkly-local-development-only PORT=8080 pnpm nx run backend:dev
```

Контейнеры продолжат обращаться к `backend:8080`, даже если профиль backend выключен. Локальный сервер должен слушать `0.0.0.0` или доступный Docker интерфейс хоста; сервер, привязанный только к `127.0.0.1`, может быть недоступен контейнерам на Linux. Для локального frontend: `pnpm nx run frontend:dev -- --host 0.0.0.0 --port 3000`.

Порты можно переопределить через `FRONTEND_PORT`, `BACKEND_PORT`, `ANALYTICS_PORT`, `ML_PORT`, `POSTGRES_PORT`, `CLICKHOUSE_HTTP_PORT`, `CLICKHOUSE_NATIVE_PORT` в окружении или корневом `.env`. Межсервисные URL должны использовать **порт хоста** после переопределения. При изменении порта frontend задайте также `FRONTEND_ORIGIN` (по умолчанию `http://localhost:3000`). Backend получает адрес БД из `POSTGRES_HOST` (в Compose — `postgres`, вне Docker — `localhost`) и `POSTGRES_PORT`.

# Sparkly

- `backend/` — Control Plane на Go.
- `frontend/` — SvelteKit.
- `agent/` — Rust.
- `ml/` — Python.
- `analytics/` — аналитика на Go.
- `docs/` — архитектура и решения.

Protobuf-контракты хранятся в каталогах `proto/` соответствующих сервисов. Задачи проектов запускаются через Nx; JavaScript-зависимости управляются pnpm.

## Разработка

Требуются Node.js 22.18+, pnpm 11.7.0 и родные инструменты проектов: Go, Rust/Cargo, Python.

```sh
pnpm install
pnpm dev                       # frontend
pnpm check                     # проверки пяти проектов и Protobuf-контрактов
pnpm build                     # frontend, backend, analytics и agent
pnpm nx run backend:dev        # запуск отдельного проекта
pnpm nx run ml:dev
pnpm graph
```

Nx использует `project.json` в каждом проекте. Кеш Nx включён для сборки Go и frontend.

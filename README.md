# Sparkly

Минимальные независимые проекты монорепозитория:

- `backend/` — Control Plane на Go.
- `frontend/` — SvelteKit, TypeScript.
- `agent/` — Rust, Cargo.
- `ml/` — Python.
- `analytics/` — аналитика на Go.
- `docs/` — архитектура и решения.

В каждом проекте есть README, `.gitignore` и пустой `proto/service.proto`. Protobuf-контракты и их генерация пока не настроены. Задачи проектов запускаются через Nx; JavaScript-зависимости управляются pnpm.

## Разработка

Требуются Node.js 22.12+, pnpm 11.7.0 и родные инструменты проектов: Go, Rust/Cargo, Python.

```sh
pnpm install
pnpm dev                       # frontend
pnpm check                     # проверки всех пяти проектов
pnpm build                     # frontend, backend, analytics и agent
pnpm nx run backend:dev        # запуск отдельного проекта
pnpm nx run ml:dev
pnpm graph
```

Nx использует `project.json` в каждом проекте. Для Python пока доступны `check` и `dev`; сборка пакета не настроена. pnpm workspace включает frontend; Go, Rust и Python сохраняют собственные манифесты и инструменты.

Кеш Nx включён для сборки Go и frontend. Проверки проектов и задачи Cargo выполняются без кеша Nx. Межсервисные зависимости задач будут добавлены вместе с генерацией Protobuf.

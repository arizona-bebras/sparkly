# Frontend

Минимальный SvelteKit-проект на TypeScript. Требуется Node.js 22.18+ и pnpm 11.7.0.

Из корня репозитория:

```sh
pnpm install
pnpm nx run frontend:dev
pnpm nx run frontend:check
pnpm nx run frontend:check:types
pnpm nx run frontend:build
pnpm nx run frontend:preview
pnpm nx run frontend:lint
pnpm nx run frontend:lint:fix
pnpm nx run frontend:format
pnpm nx run frontend:format:check
```

Локальные команды: `pnpm dev`, `pnpm check`, `pnpm build`, `pnpm preview`.

Nx запускает команды Vite Plus напрямую. `frontend:check` объединяет задачи `lint`, `format:check` и `check:types`; `frontend:build` зависит от `check`, а `frontend:preview` — от `build`. Lint, проверка форматирования и сборка кешируются Nx. Задачи с исправлением файлов (`lint:fix`, `format`) выполняются без кеша; `dev` и `preview` отмечены как непрерывные задачи.

Vite Plus используется для сервера разработки (`vp dev`), production-сборки (`vp build`), предпросмотра (`vp preview`), lint и форматирования. Команды запускаются через локальную зависимость, глобальная установка `vp` не требуется. Версии `vite-plus` и встроенного Vite закреплены в каталоге `pnpm-workspace.yaml`; alias и override обеспечивают одну версию Vite для SvelteKit и плагинов.

`pnpm lint` запускает Oxlint и завершает проверку с ошибкой при предупреждениях, `pnpm lint:fix` исправляет доступные нарушения, `pnpm format` форматирует файлы, `pnpm format:check` проверяет форматирование без записи. Настройки находятся в блоках `lint` и `fmt` в `vite.config.ts`; сгенерированные `.svelte-kit/`, `build/` и `dist/` исключены. Oxlint проверяет скрипты в `.svelte`, а проверка шаблонов и типов остаётся за `svelte-check`.

`pnpm check` выполняет lint, проверку форматирования и `pnpm check:types` (Svelte и TypeScript). Эти проверки также запускаются перед сборкой.

`src/routes/+page.svelte` — главная страница, `src/app.html` — HTML-шаблон. `svelte-kit sync` генерирует конфигурацию и типы в `.svelte-kit/` при установке зависимостей и проверке проекта.

Используется `adapter-node`: результат сборки находится в `build/`, запуск из каталога frontend — `node build`. Конфигурация SvelteKit хранится в стандартном `svelte.config.js`; код приложения и Vite-конфигурация используют TypeScript.

Protobuf-контракты и браузерный транспорт до backend пока не настроены.

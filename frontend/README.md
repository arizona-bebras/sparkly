# Frontend

Минимальный SvelteKit-проект на TypeScript. Требуется Node.js 22.12+ и pnpm 11.7.0.

Из корня репозитория:

```sh
pnpm install
pnpm nx run frontend:dev
pnpm nx run frontend:check
pnpm nx run frontend:build
pnpm nx run frontend:preview
```

Локальные команды: `pnpm dev`, `pnpm check`, `pnpm build`, `pnpm preview`.

`src/routes/+page.svelte` — главная страница, `src/app.html` — HTML-шаблон. `svelte-kit sync` генерирует конфигурацию и типы в `.svelte-kit/` при установке зависимостей и проверке проекта.

Используется `adapter-node`: результат сборки находится в `build/`, запуск из каталога frontend — `node build`. Конфигурация SvelteKit хранится в стандартном `svelte.config.js`; код приложения и Vite-конфигурация используют TypeScript.

Protobuf-контракты и браузерный транспорт до backend пока не настроены.

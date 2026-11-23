# Frontend Guidelines & Architecture (Sparkly)

Инструкции и правила разработки клиентской части Sparkly (`frontend/`).
Этот документ автоматически загружается агентом при работе в директории `frontend/`.

---

## 1. Стек технологий

- **Фреймворк**: SvelteKit 2 + Svelte 5 (строго синтаксис Runes).
- **Стилизация**: Tailwind CSS 4 (`@tailwindcss/vite`), tailwind-variants, tw-animate-css.
- **UI-компоненты**: shadcn-svelte (порт компонентов на Bits UI 2).
- **Иконки**: `@lucide/svelte`.
- **Серверное состояние и кэш**: `@tanstack/svelte-query` v6.
- **Таблицы**: `@tanstack/svelte-table` v9.
- **Формы и валидация**: `sveltekit-superforms` v2 (в режиме SPA), Zod 4 адаптер (`zod4`), `formsnap`.
- **API Транспорт**: ConnectRPC (`@connectrpc/connect`, `@connectrpc/connect-web`), Protobuf (`@bufbuild/protobuf`).
- **Тулчейн и линтинг**: Vite 8 beta (`vp` / Vite Plus), Biome (`vp lint`, `vp fmt`), TypeScript 5.9, `svelte-check`.

---

## 2. Структура директорий

```
frontend/
├── proto/                          # ConnectRPC клиент и сгенерированные Protobuf типы
│   ├── client.ts                   # Экспорт инстанса client = { user, park, device, issue, build, release }
│   └── gen/                        # Сгенерированные .ts файлы из backend/proto/service.proto
├── src/
│   ├── app.html                    # Базовый HTML-шаблон
│   ├── app.css                     # Глобальные стили Tailwind 4 и переменные темы
│   ├── lib/
│   │   ├── form-spa-util.svelte.ts # Утилита spaForm() для работы с Superforms в режиме SPA
│   │   ├── utils.ts                # Хелпер cn() для объединения классов
│   │   └── components/
│   │       ├── ui/                 # Атомарные UI-компоненты (shadcn / Bits UI)
│   │       ├── forms/              # Формы ввода данных (схема schema.ts + компонент .svelte)
│   │       │   ├── SubmitBtn.svelte
│   │       │   ├── group/
│   │       │   ├── park/
│   │       │   ├── release/
│   │       │   ├── signin/
│   │       │   └── signup/
│   │       └── entities/           # Доменно-изолированные компоненты сущностей
│   │           ├── device/         # Устройства (DeviceList, DeviceFilters, columns, cells)
│   │           ├── group/          # Группы устройств (GroupList, ...)
│   │           ├── issue/          # Проблемы и трейсы (IssueList, IssueFilters, IssueDetails)
│   │           ├── build/          # Сборки (BuildList, BuildFilters, BuildDetailsDialog, columns, cells)
│   │           ├── release/        # Релизы (ReleaseTimeline, ReleaseFilters)
│   │           ├── park/           # Парки (ParkList, ParkDialog)
│   │           └── sidebar/        # Боковое меню и профиль (app-sidebar, nav-user)
│   └── routes/                     # Страницы маршрутов SvelteKit
│       ├── +layout.svelte          # Главный лейаут (QueryClientProvider, тема)
│       ├── +page.svelte            # Корневая страница (редирект или выбор парка)
│       ├── auth/                   # Маршруты аутентификации (/signin, /signup)
│       └── control-panel/          # Панель управления парком
│           ├── +layout.svelte      # Лейаут с AppSidebar и навигацией
│           └── [park_id]/          # Разделы конкретного парка:
│               ├── +page.svelte    # Дашборд парка
│               ├── devices/        # /control-panel/[park_id]/devices
│               ├── groups/         # /control-panel/[park_id]/groups
│               ├── issues/         # /control-panel/[park_id]/issues (+ [issue_id])
│               ├── builds/         # /control-panel/[park_id]/builds
│               ├── releases/       # /control-panel/[park_id]/releases
│               └── settings/       # /control-panel/[park_id]/settings
```

---

## 3. Архитектурный принцип: Тонкие страницы маршрутов

Файлы `routes/.../+page.svelte` должны оставаться **минималистичными** (до 15–20 строк кода).
Они служат только точками входа SvelteKit:

1. Получают параметры маршрута через `page.params` (например, `parkId = $derived(page.params.park_id ?? '')`).
2. Задают заголовок страницы через `<svelte:head><title>...</title></svelte:head>`.
3. Отображают заголовок раздела и монтируют соответствующий компонент из `$lib/components/entities/<entity>/`.

Вся логика запросов, мутаций, модалок, фильтров и таблиц инкапсулируется внутри `src/lib/components/entities/<entity>/`.

---

## 4. Правила работы с формами (SPA + `spaForm`)

Все формы в проекте работают на клиенте (SPA-режим) без серверных экшенов SvelteKit (`+page.server.ts`).

### Паттерн создания формы:

1. Каждая форма имеет отдельную папку в `$lib/components/forms/<name>/` с файлами:
   - `schema.ts` — схема валидации Zod (`formSchema = z.object({ ... })`).
   - `<Name>.svelte` — компонент формы.
2. В компоненте формы используется утилита `spaForm`:
   ```svelte
   <script lang="ts">
     import { spaForm } from '$lib/form-spa-util.svelte';
     import * as Form from '$lib/components/ui/form/index.js';
     import { Input } from '$lib/components/ui/input/index.js';
     import SubmitBtn from '$lib/components/forms/SubmitBtn.svelte';
     import { formSchema } from './schema.js';

     let { parkId, onSuccess } = $props();

     const { form, formData, isValid, enhance } = spaForm(formSchema, {
       async onUpdated({ form }) {
         if (form.valid) {
           await mutation.mutateAsync(form.data);
           onSuccess?.();
         }
       },
     });
   </script>

   <form method="POST" use:enhance class="flex flex-col gap-4">
     <Form.Field {form} name="name">
       <Form.Control>
         {#snippet children({ props })}
           <Input {...props} bind:value={$formData.name} placeholder="Название" />
         {/snippet}
       </Form.Control>
       <Form.FieldErrors />
     </Form.Field>

     <SubmitBtn
       {isValid}
       text="Сохранить"
       loadingText="Сохранение..."
       isPending={mutation.isPending}
     />
   </form>
   ```
3. Кнопка отправки всегда использует `<SubmitBtn {isValid} ... />`, которая автоматически блокируется, если форма невалидна или находится в состоянии отправки (`isPending`).

---

## 5. Таблицы (`DataTable`) и Безопасность (XSS)

Таблицы строятся на базе компонента `$lib/components/ui/data-table/data-table.svelte` (`@tanstack/svelte-table`).

### Строгие правила таблиц:

1. **Защита от XSS (Категорический запрет `createRawSnippet`)**:
   - Никогда не использовать `createRawSnippet` с HTML-строками для ячеек таблицы! Данные из API могут содержать вредоносный код.
   - Все нетривиальные ячейки (названия со ссылками, бейджи, кнопки копирования, даты, действия) **всегда** реализуются как отдельные типизированные `.svelte` компоненты в папке `cells/` и подключаются через `renderComponent`:
     ```ts
     columnHelper.accessor('name', {
       header: 'Устройство',
       cell: ({ row }) =>
         renderComponent(DeviceNameCell, {
           name: row.original.name,
           id: row.original.id,
         }),
     });
     ```
2. **Структура колонок**:
   - Описание колонок выносится в отдельный файл `columns.ts` рядом со списком.
   - При необходимости интерактивных действий из ячеек (просмотр деталей, удаление) экспортируется фабричная функция:
     `export function createColumns(callbacks: { onViewDetails?: (item) => void; onDelete?: (item) => void })`.
3. **Тулбар и фильтрация**:
   - Тулбар таблицы подключается через сниппет `{#snippet filterToolbar()}` и передаётся пропом `toolbar={filterToolbar}` в `DataTable`.

---

## 6. Правила использования Debounce в инпутах

Debounce предназначен для ограничения частоты сетевых запросов при вводе текста пользователем.

### Когда ОБЯЗАТЕЛЕН `DebouncedInput` (задержка 350 мс):

- В строках поиска и текстовых фильтрах, изменение которых **отправляет RPC/HTTP-запрос на сервер** (например, поиск по названию в `ReleaseTimeline`, поиск по коммиту в `BuildFilters`, поиск по устройствам в `DeviceList`).
- Использовать: `import { DebouncedInput } from '$lib/components/ui/debounced-input/index.js';`.

### Когда используется обычный `Input` (БЕЗ debounce):

- В формах создания и редактирования (`spaForm` / `Form.Field`), так как форма отправляется только по явному нажатию кнопки Submit.
- В чисто локальных клиентских полях ввода, где сетевые запросы не инициируются, чтобы не создавать искусственной задержки для пользователя.

---

## 7. Работа с API и TanStack Query

1. **ConnectRPC клиент**:
   - Все методы вызываются через объект `client` из `$proto`:
     `client.device.*`, `client.build.*`, `client.release.*`, `client.issue.*`, `client.park.*`, `client.user.*`.
2. **Составные queryKey**:
   - В ключ запроса TanStack Query обязательно включаются все параметры фильтрации:
     `queryKey: ['builds', parkId, versionParam, commitParam, fromDateStr, toDateStr]`.
   - Это гарантирует корректное кэширование и предотвращает коллизии между разными страницами и парками.
3. **Обработка дат и временных интервалов**:
   - Для выбора периодов дат всегда используется компонент `DateRangePicker` (`bind:value={dateRange}`).
   - Перевод диапазона в Protobuf Timestamp выполняется через `@internationalized/date` и `@bufbuild/protobuf/wkt`:
     ```ts
     const startDate = dateRange.start.toDate(getLocalTimeZone());
     startDate.setHours(0, 0, 0, 0);
     const fromTime = timestampFromDate(startDate);

     const endDate = (dateRange.end ?? dateRange.start).toDate(getLocalTimeZone());
     endDate.setHours(23, 59, 59, 999);
     const toTime = timestampFromDate(endDate);
     ```
4. **Индикация загрузки**:
   - При первой загрузке (когда данных ещё нет) отображать скелетоны `<Skeleton class="..." />`.
   - При смене фильтров использовать `placeholderData: keepPreviousData` и отображать полупрозрачный оверлей с индикатором `isPlaceholderData`.
   - При ошибке рендерить блок с сообщением и кнопкой «Повторить» (`refetch()`).
5. **Мутации и инвалидация кэша**:
   - При создании, обновлении или удалении сущностей инвалидировать соответствующий префикс ключа:
     `queryClient.invalidateQueries({ queryKey: ['builds', parkId] })`.

---

## 8. Svelte 5: Только Runes

- Использовать строго современный синтаксис Svelte 5:
  - Состояние: `$state(...)`.
  - Вычисляемые значения: `$derived(...)` или `$derived.by(() => ...)`.
  - Пропсы: `let { prop1, prop2 = defaultValue }: Props = $props();`.
  - Двустороннее связывание пропсов: `let { value = $bindable() }: Props = $props();`.
  - Побочные эффекты: `$effect(() => { ... })`.
- **Категорически запрещён legacy синтаксис Svelte 3/4**:
  - Запрещено `export let`.
  - Запрещено реактивное объявление `$:`.
  - Запрещены `<slot />` (использовать сниппеты `{#snippet ...}` и `children` проп).

---

## 9. ЖЁСТКОЕ ПРАВИЛО: НИКАКИХ КОММЕНТАРИЕВ В КОДЕ

В любом генерируемом или редактируемом коде (Svelte, TypeScript, JavaScript, CSS) **полностью запрещены любые комментарии**:

- Ни однострочных (`// ...`),
- Ни многострочных (`/* ... */`),
- Ни HTML-комментариев (`<!-- ... -->`).

Код должен быть самодокументируемым за счёт понятных названий функций, переменных и строгой типизации TypeScript.

---

## 10. Проверка качества перед завершением задачи

После любых правок фронтенда агент обязан выполнить проверку:

1. Форматирование: `pnpm --dir frontend run fmt`
2. Линтинг и проверка типов: `pnpm --dir frontend run check` (включает `vp lint`, `vp fmt:check`, `svelte-check`, `tsc`).
   Команда должна завершаться с кодом `0` (0 errors, 0 warnings).

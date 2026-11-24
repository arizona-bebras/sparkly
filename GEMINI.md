# Sparkly Workspace Rules & Architecture

Sparkly — платформа управления парком IoT/embedded устройств, сборок прошивок, релизов и мониторинга инцидентов.

## Архитектура системы
- **Backend (`backend/`)**: Go, GORM, PostgreSQL, ClickHouse, RabbitMQ, Step CA (PKI). ConnectRPC API сервис (`sparkly.backend.v1.*`).
- **Envoy Proxy (`envoy/`)**: gRPC-Web и HTTP проксирование запросов к бэкенду.
- **Frontend (`frontend/`)**: SvelteKit 2 + Svelte 5 (Runes), Tailwind CSS 4, TanStack Query/Table.
- **API Контракты (`backend/proto/service.proto`)**: единый источник правды для Protobuf сообщений и ConnectRPC сервисов.

## Глобальные правила (Global Strict Rules)
1. **Категорический запрет на комментарии**: в любом коде (Go, Svelte, TypeScript, SQL, HTML, CSS) полностью запрещены комментарии (`//`, `/* */`, `<!-- -->`).
2. **Язык общения**: все ответы пользователю формулируются на русском языке. Все файлы проекта должны иметь кликабельные ссылки формата `file:///...` с прямыми слэшами.
3. **Безопасность**: защита от XSS, строгая валидация входных данных через Protobuf (`buf.validate`).

## Правила подсистем
@[Frontend Guidelines](frontend/AGENTS.md)

# ML

Из корня репозитория:

```sh
uv sync --project ml --locked
pnpm nx run ml:check
pnpm nx run ml:lint:fix
pnpm nx run ml:format
```

Требуются uv и Python 3.11+. Nx запускает команды через `uv run --locked`: uv создаёт окружение `ml/.venv` и синхронизирует зависимости автоматически, активировать окружение вручную не нужно.

`check` запускает Ruff lint, проверку форматирования и компиляцию Python для проверки синтаксиса. Настройки и dev-зависимости находятся в `pyproject.toml`, разрешённые версии закреплены в `uv.lock`. `pnpm nx run ml:dev` запускает установленную команду `sparkly-ml`.

Для добавления зависимости используйте `uv add --project ml <package>`, для инструмента разработки — `uv add --project ml --dev <package>`. После изменения зависимостей обновляйте и коммитьте `pyproject.toml` вместе с `uv.lock`.

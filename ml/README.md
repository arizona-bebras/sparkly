# ML

Минимальный Python-проект с layout `src/`. ML-зависимости и интеграция OpenRouter пока не добавлены.

Из этого каталога:

```sh
python3 -m venv .venv
.venv/bin/python -m pip install -e .
.venv/bin/sparkly-ml
```

Без установки: `PYTHONPATH=src python3 -m sparkly_ml`.

`proto/service.proto` — пустая заготовка контракта.

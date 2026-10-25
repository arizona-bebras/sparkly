FROM python:3.11-slim-bookworm AS build
COPY --from=ghcr.io/astral-sh/uv:0.9.5 /uv /usr/local/bin/uv
WORKDIR /app
ENV UV_PYTHON_DOWNLOADS=never UV_LINK_MODE=copy
COPY pyproject.toml uv.lock README.md ./
COPY src ./src
RUN uv sync --locked --no-dev --no-editable

FROM python:3.11-slim-bookworm
WORKDIR /app
COPY --from=build /app/.venv /app/.venv
ENV PATH="/app/.venv/bin:$PATH" PORT=8082 PYTHONUNBUFFERED=1
USER 65532:65532
EXPOSE 8082
ENTRYPOINT ["sparkly-ml"]

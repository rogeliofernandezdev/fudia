FROM python:3.12.14-slim

ENV PYTHONDONTWRITEBYTECODE=1
ENV PYTHONUNBUFFERED=1

WORKDIR /app

RUN groupadd --system fudia \
    && useradd --uid 10001 --gid fudia --create-home fudia

COPY pyproject.toml README.md constraints.txt ./
COPY config ./config
COPY src ./src
COPY prompts ./prompts

RUN python -m pip install --no-cache-dir \
    -c constraints.txt .

RUN chown -R fudia:fudia /app
USER 10001:10001

EXPOSE 8010

HEALTHCHECK --interval=30s --timeout=3s --start-period=20s --retries=3 \
  CMD python -c "import urllib.request; urllib.request.urlopen('http://127.0.0.1:8010/live', timeout=2)" || exit 1

CMD ["uvicorn","src.main:app","--host","0.0.0.0","--port","8010"]

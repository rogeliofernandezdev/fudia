FROM python:3.12-slim

ENV PYTHONDONTWRITEBYTECODE=1
ENV PYTHONUNBUFFERED=1

WORKDIR /app
COPY pyproject.toml README.md ./
COPY config ./config
COPY src ./src
COPY prompts ./prompts

RUN python -m pip install --no-cache-dir .

EXPOSE 8010
CMD ["uvicorn","src.main:app","--host","0.0.0.0","--port","8010"]

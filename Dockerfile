# --- Этап 1: Сборка приложения ---
FROM golang:1.24 AS builder

# Устанавливаем рабочую директорию внутри контейнера
WORKDIR /app

# Копируем модули и скачиваем зависимости (кэшируем на будущее)
COPY go.mod go.sum ./
RUN go mod download

# Копируем весь исходный код в контейнер
COPY . .

# Переходим в директорию с точкой входа команды
WORKDIR /app/cmd

# Собираем статический бинарник для Linux (без CGO)
RUN CGO_ENABLED=0 GOOS=linux go build -o /app/bin/app main.go

# --- Этап 2: Формирование финального образа ---
FROM debian:bullseye-slim

# Обновляем индекс пакетов и устанавливаем ffmpeg
RUN apt-get update \
    && apt-get install -y --no-install-recommends \
    ffmpeg \
    ca-certificates \
    && rm -rf /var/lib/apt/lists/*

# Создаём пользователя (неroot) для запуска приложения (рекомендуется в продакшн)
RUN useradd --user-group --create-home --shell /bin/false appuser

# Копируем собранный бинарник из этапа builder
COPY --from=builder /app/bin/app /usr/local/bin/app

# Назначаем права на запуск для пользователя
RUN chown appuser:appuser /usr/local/bin/app \
    && chmod 755 /usr/local/bin/app

# Переключаемся на непользовательского пользователя
USER appuser

# (Опционально) если ваше приложение слушает определённый порт, можно указать expose:
# EXPOSE 8080

# По умолчанию контейнер будет запускать бинарник
ENTRYPOINT ["app"]
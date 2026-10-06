# Шаг 1: Компиляция
FROM golang:1.27-alpine AS builder

# Устанавливаем системные пакеты для сборки
RUN apk add --no-cache git

WORKDIR /build

# Копируем файлы конфигурации модуля
COPY go.mod go.sum ./

# Копируем исходный код
COPY main.go ./

# ИСПРАВЛЕНО: Автоматически чистим зависимости и наполняем go.sum
RUN go mod tidy && go mod download

# Собираем изолированный бинарник
RUN CGO_ENABLED=0 GOOS=linux go build -o /app-bin main.go

# Шаг 2: Финальный продакшен-контейнер
FROM alpine:latest
RUN apk --no-cache add ca-certificates

WORKDIR /app
COPY --from=builder /app-bin .

# Сервер слушает порт 8000
EXPOSE 8000

CMD ["./app-bin"]

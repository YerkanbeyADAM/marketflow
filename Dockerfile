FROM golang:1.23

WORKDIR /app

# Установим git (если go mod требует)
RUN apt-get update && apt-get install -y git

# Копируем go.mod и go.sum, скачиваем зависимости
COPY go.mod ./
COPY go.sum ./
RUN go mod download

# Копируем остальной код
COPY . .

# Собираем приложение
RUN go build -o marketflow ./cmd/app

# Запускаем приложение напрямую (без wait-for-it.sh)
CMD ["/app/marketflow"]

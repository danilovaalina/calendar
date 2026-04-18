# Стейдж сборки
FROM golang:1.25-alpine AS builder
WORKDIR /app
COPY go.mod go.sum ./
RUN go mod download
COPY . .

RUN go build -o calendar cmd/main.go

# Стейдж запуска
FROM alpine:latest
WORKDIR /root/

# Копируем бинарник и веб
COPY --from=builder /app/calendar .

CMD ["./calendar"]

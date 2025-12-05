FROM golang:1.23-alpine AS builder

# нужен git для скачивания модулей
RUN apk add --no-cache git

WORKDIR /app

# сначала go.mod (go.sum может и не быть, не страшно)
COPY go.mod ./

# попробуем заранее скачать зависимости (если go.sum нет — он создастся)
RUN go mod download || true

# теперь весь проект
COPY . .

# СБОРКА:
# -mod=mod разрешает go самому дописать go.sum и скачать недостающие модули
RUN go build -mod=mod -o server ./cmd/server

FROM alpine:3.20

WORKDIR /app
COPY --from=builder /app/server ./server
COPY web ./web
COPY .env .env

EXPOSE 8080

CMD ["./server"]

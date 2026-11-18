FROM golang:1.27-alpine AS builder

WORKDIR /src

COPY go.mod go.sum ./
RUN go mod download

COPY . .
RUN CGO_ENABLED=0 GOOS=linux go build -trimpath -o /out/stats-service ./backend/stats-service

FROM alpine:3.24

WORKDIR /app

COPY --from=builder /out/stats-service ./stats-service

ENTRYPOINT ["./stats-service"]

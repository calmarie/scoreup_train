FROM golang:1.27-alpine AS builder

WORKDIR /src

COPY go.mod go.sum ./
RUN go mod download

COPY . .
RUN CGO_ENABLED=0 GOOS=linux go build -trimpath -o /out/user-service ./backend/user-service

FROM alpine:3.24

WORKDIR /app

COPY --from=builder /out/user-service ./user-service

EXPOSE 50053

ENTRYPOINT ["./user-service"]

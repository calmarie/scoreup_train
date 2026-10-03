##binary creation

FROM golang:1.27-alpine AS builder

WORKDIR /app

COPY go.mod ./
COPY go.sum ./

RUN go mod download

COPY . .

RUN CGO_ENABLED=0 GOOS=linux go build -o scoreup_train .


##execution from binary

FROM alpine:3.24

COPY --from=builder /app/scoreup_train .

EXPOSE 8080

ENTRYPOINT ["./scoreup_train"]



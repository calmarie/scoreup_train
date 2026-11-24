# ScoreUp Train

Мини-апп для тренировки на задачах. Пользователь получает задачу, отправляет ответ и видит XP, серии правильных ответов и статистику по дням и темам.
(Сейчас в разработке, доступна только backend часть. Далее описана реализованная на данный момент инфраструктура.)

## Стек технологий

- **Go и `net/http`** — бэкенд и HTTP/JSON API в `api-gateway`.
- **gRPC, Protocol Buffers и TLS** — типизированные контракты и защищённые вызовы между сервисами.
- **PostgreSQL, `pgx` и `golang-migrate`** — хранение пользователей, задач, ответов и статистики, подключение к БД и миграции схемы.
- **Redis и `go-redis`** — хранение хешей refresh-токенов.
- **Apache Kafka и `franz-go`** — передача событий о попытках решения для асинхронного обновления статистики.
- **JWT с подписью Ed25519** — авторизация запросов к защищённым маршрутам.
- **Docker и Docker Compose** — сборка и локальный запуск сервисов и их зависимостей.
- **OpenAPI 3.1 и Swagger UI** — спецификация и интерактивная документация HTTP API.

## Сервисы

| Сервис | За что отвечает |
| --- | --- |
| `api-gateway` | Принимает HTTP-запросы, проверяет access-токен на защищённых маршрутах и вызывает остальные сервисы по gRPC. |
| `user-service` | Хранит пользователей в PostgreSQL, подписывает access-токены и хранит хеши refresh-токенов в Redis. |
| `training-service` | Выдаёт активные задачи, проверяет и сохраняет ответы в PostgreSQL, публикует попытки в Kafka. |
| `stats-service` | Читает события из Kafka и обновляет общую, дневную и тематическую статистику в PostgreSQL. |

После отправки ответа `training-service` публикует событие в Kafka-топик `question-solved`. `stats-service` обновляет все три вида статистики в одной транзакции и использует ID ответа, чтобы добиться консистентности данных. XP начисляется за первое правильное решение задачи. Статистика обновляется асинхронно, а дни считаются по UTC.

## Локальный запуск

Команды ниже рассчитаны на Bash в Linux. Для запуска понадобятся Docker и Compose, для создания сертификатов и ключей — OpenSSL 3. В примерах HTTP-запросов используются `curl` и `jq`. Для  HTTP-запросов можете также использовать Postman.

В корне проекта создайте `.env` на основе примера:

```bash
if [ ! -f .env ]; then
    cp .env.example .env
fi
```

Если в `certs/` ещё нет локальных сертификатов и ключей, создайте их следующими командами. Внутренний gRPC использует TLS. Ключи `private.pem` и `public.pem` используются для JWT.

```bash
mkdir -p certs
openssl req -x509 -newkey rsa:2048 -nodes -days 365 -subj '/CN=ScoreUp Local CA' -addext 'basicConstraints=critical,CA:TRUE' -keyout certs/ca.key -out certs/ca.pem

for name in user training stats; do
    service="${name}-service"
    openssl req -new -newkey rsa:2048 -nodes -subj "/CN=$service" -addext "subjectAltName=DNS:$service,DNS:localhost" -keyout "certs/$name.key" -out "certs/$name.csr"
    openssl x509 -req -in "certs/$name.csr" -CA certs/ca.pem -CAkey certs/ca.key -CAcreateserial -days 365 -copy_extensions copy -out "certs/$name.crt"
done

openssl genpkey -algorithm ED25519 -out certs/private.pem
openssl pkey -in certs/private.pem -pubout -out certs/public.pem
```

Запустите сервисы и проверьте gateway:

```bash
docker compose up -d --build
docker compose ps
curl -fsS -i http://localhost:8080/health
```

Compose поднимает PostgreSQL, Redis и Kafka, применяет миграции и создаёт Kafka-топик. Если в `.env` изменён `HTTP_PORT`, подставьте его вместо `8080`.

## HTTP API

Чтобы посмотреть документацию в Swagger UI, запустите из корня проекта:

```bash
python3 -m http.server 8081 --bind 127.0.0.1 --directory docs
```

Откройте [http://127.0.0.1:8081/swagger.html](http://127.0.0.1:8081/swagger.html). Страница читает [спецификацию OpenAPI](docs/openapi.json) и загружает Swagger UI из интернета. Для отправки запросов можно импортировать этот же JSON в Postman.

На маршрутах задач и статистики передавайте `Authorization: Bearer <access_token>`. Gateway берёт ID пользователя из токена.

| Метод и путь | Что делает |
| --- | --- |
| `GET /health` | Проверяет, что gateway отвечает. |
| `POST /api/v1/users/auth/register/{telegramID}` | Регистрирует пользователя или возвращает сессию существующего. |
| `POST /api/v1/users/auth/refresh/{telegramID}` | Принимает `{"refresh_token":"..."}` и обновляет сессию. |
| `GET /api/v1/users/telegram/{telegramID}` | Ищет пользователя по Telegram ID. |
| `GET /api/v1/question` | Возвращает случайную активную задачу. |
| `POST /api/v1/question/answer` | Принимает `question_id` и `answer`, возвращает результат проверки. |
| `GET /api/v1/stats/stats` | Общая статистика. |
| `GET /api/v1/stats/xp` | Суммарный XP. |
| `GET /api/v1/stats/streak` | Текущая серия дней с правильными ответами. |
| `GET /api/v1/stats/correctstreak` | Текущая серия правильных ответов подряд. |
| `GET /api/v1/stats/daily/{date}` | Статистика за дату в формате `YYYY-MM-DD`. |
| `GET /api/v1/stats/weekly` | Статистика за сегодня и шесть предыдущих дней. Дни без попыток пропускаются. |
| `GET /api/v1/stats/topics/{topic}` | Статистика по одной теме. |
| `GET /api/v1/stats/topics` | Статистика по всем темам пользователя. |

Регистрация и повторная выдача сессии при недействительном refresh-токене пока доверяют Telegram ID из запроса. Этот режим предназначен для локальной разработки. Для публичного запуска нужна проверка личности через Telegram.

## Пример: ответить на задачу и посмотреть статистику

В свежей базе задач нет. Добавьте пример в PostgreSQL сервиса тренировок:

```bash
docker compose exec -T training-postgres sh -c 'exec psql -v ON_ERROR_STOP=1 -U "$POSTGRES_USER" -d "$POSTGRES_DB"' <<'SQL'
INSERT INTO questions (type, difficulty, topic, text, options, correct_answer, xp)
VALUES ('radio', 1, 'binary_numbers', 'What is 1011 in decimal?',
        '["9","10","11","13"]'::jsonb, '"11"'::jsonb, 20)
RETURNING id;
SQL
```

Подставьте полученный `id` вместо `1`:

```bash
base_url='http://localhost:8080'
session=$(curl -fsS -X POST "$base_url/api/v1/users/auth/register/123")
access_token=$(jq -er '.access_token' <<< "$session")
auth="Authorization: Bearer $access_token"

curl -fsS -H "$auth" "$base_url/api/v1/question"
curl -fsS -H "$auth" -H 'Content-Type: application/json' \
    --data '{"question_id":1,"answer":"11"}' "$base_url/api/v1/question/answer"
curl -fsS -H "$auth" "$base_url/api/v1/stats/topics/binary_numbers"
```

Если статистика сразу вернула `404`, повторите запрос после того, как `stats-service` обработает событие из Kafka. Ответ `"11"` передаётся как JSON-строка, поскольку проверка сравнивает JSON побайтно.

## Где искать код

- [HTTP-обработчики и JWT middleware](backend/api-gateway/)
- [Пользователи и сессии](backend/user-service/)
- [Задачи и ответы](backend/training-service/)
- [Статистика и чтение Kafka](backend/stats-service/)
- [Контракты gRPC](backend/proto/) и [сгенерированный по proto файлам Go-код](backend/gen/)

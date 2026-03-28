# users-api — crosscheck example project

A minimal Node.js/Express REST API backed by **MariaDB** and **Redis**, used to demonstrate a full [crosscheck](https://github.com/lozymon/crosscheck) test suite.

## What it does

| Endpoint | Description |
|---|---|
| `POST /users` | Creates a user (persists to MariaDB, caches in Redis, fires a webhook) |
| `GET /users/:id` | Returns a user by id |

## Prerequisites

- [Docker](https://docs.docker.com/get-docker/) + Docker Compose
- [crosscheck](https://github.com/lozymon/crosscheck) (`cx`) installed

## Run the stack

```bash
cd examples/users-api

# Build and start all services (MariaDB, Redis, app)
docker compose up --build -d

# Wait for the app to be ready (first run pulls images)
docker compose logs -f app
```

The API is available at `http://localhost:3000`.

## Run the tests

The test suite starts a local mock server on a random port and passes its URL to
the app as `WEBHOOK_URL` so outbound webhook calls are captured.

```bash
# From the repo root
WEBHOOK_URL=$(cx run examples/users-api/tests/ --reporter pretty 2>&1 | grep MOCK_URL || true)
cx run examples/users-api/tests/ \
  --env BASE_URL=http://localhost:3000 \
  --reporter pretty
```

Or simply (crosscheck reads `BASE_URL` from your shell):

```bash
export BASE_URL=http://localhost:3000
cx run examples/users-api/tests/
```

The test suite:
1. **Creates a user** — asserts HTTP 201, captures `userId`
2. **Checks MariaDB** — asserts the `users` row was inserted (`adapter: mysql`)
3. **Checks Redis** — asserts the `user:<id>` key was cached (`adapter: redis`)
4. **Checks the webhook** — asserts the mock server received `POST /` with the right payload (`adapter: mock`)
5. **Fetches the user** — asserts HTTP 200 with correct fields
6. **Fetches an unknown user** — asserts HTTP 404

## Tear down

```bash
docker compose down -v
```

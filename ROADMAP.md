# crosscheck — Roadmap

> Build tracker. See [PLAN.md](PLAN.md) for design decisions and architecture rationale.

## MVP

### Project setup

- [x] Go module (`github.com/lozymon/crosscheck`)
- [x] Directory structure
- [x] Core dependencies (`cobra`, `yaml.v3`, `godotenv`, `fatih/color`)
- [x] `main.go` entry point
- [x] `PLAN.md` + `ROADMAP.md`
- [x] `examples/subscription-flow.cx.yaml`

### CLI commands (stubs in place, implementation pending)

- [x] `cx run` — flags: `--env-file`, `--env`, `--filter`, `--insecure`, `--output-file`, `--reporter`, `--watch`
- [x] `cx validate`
- [x] `cx init`
- [x] `cx explain` — `--ai` flag stub

### Config

- [x] YAML schema structs (`config/types.go`)
- [x] YAML parser + `version:` check + missing version warning (`config/parser.go`)

### Env + interpolation

- [x] `.env` loading + priority merge: CLI > shell > `.env` > yaml (`env/env.go`)
- [x] `{{ VAR }}` flat interpolation (`interpolate/interpolate.go`)

### HTTP client

- [x] Request builder (method, URL, headers, body)
- [x] Interpolation applied to all request fields
- [x] Response capture (`{{ capture: varName }}` → stored in vars)
- [x] `--insecure` TLS skip

### Auth

- [x] `type: static` — inject header from env var
- [x] `type: login` — POST login endpoint, capture token, inject header
- [x] Auth runs once before all tests, token available in all subsequent requests

### Assertions

- [x] HTTP response status assertion
- [x] HTTP response body assertion (exact value, partial match)
- [x] HTTP response header assertion
- [x] JSONPath assertion (`$.items[0].name`)
- [x] Regex assertion

### Postgres adapter

- [x] Connect via `pgx/v5`
- [x] Named param rewriting (`:varName` → `$1`)
- [x] Execute query + return rows as `[]map[string]any`
- [x] Row assertion against `expect:` block
- [x] `wait_for` polling (`timeout` + `interval`)
- [x] Disconnect / cleanup

### Setup / teardown hooks

- [x] File-level `setup` runs before all tests
- [x] File-level `teardown` runs after all tests (even on failure)
- [x] Per-test `setup` / `teardown`
- [x] Shell command execution with env vars available

### Variable chaining

- [x] Capture vars from response body (`{{ capture: varName }}`)
- [x] Captured vars merged into interpolation namespace
- [x] Vars available across all subsequent steps in the file

### Reporter

- [x] Pretty CLI output (colored pass/fail per test)
- [x] Structured JSON failure output (request, response, expected, actual)
- [x] `--output-file` writes JSON results alongside pretty output
- [x] Exit codes: `0` pass / `1` fail / `2` config error / `3` connection error

### File discovery

- [x] `cx run` with no args finds `**/*.cx.yaml` recursively
- [x] `cx run ./path/` finds `**/*.cx.yaml` in given directory
- [x] `cx run ./file.yaml` runs specific file (any `.yaml` accepted)
- [x] `--filter` pattern matching against test names

### Commands implementation

- [x] `cx validate` — parse + schema check, no requests fired
- [x] `cx init` — scaffold commented `crosscheck.cx.yaml` with schema hint
- [x] `cx explain` — static plain-English summary of a test file

### Schema

- [x] `crosscheck.schema.json` published alongside binary

### Distribution

- [x] `.goreleaser.yaml` config
- [x] GitHub Actions release workflow (triggers on git tag)
- [ ] Homebrew tap (`lozymon/homebrew-tap`)
- [x] `install.sh` curl script

---

## Phase 2

### Adapters

- [x] Redis adapter (`go-redis/v9`)
- [x] MySQL adapter (`go-sql-driver`)
- [x] MongoDB adapter (`mongo-driver`)
- [x] SQS adapter (`aws-sdk-go-v2`) — assert messages published to queue
- [x] SNS adapter (`aws-sdk-go-v2`) — assert notifications published
- [x] S3 adapter (`aws-sdk-go-v2`) — assert objects written to bucket
- [x] DynamoDB adapter (`aws-sdk-go-v2`) — assert items written/updated
- [x] Lambda adapter (`aws-sdk-go-v2`) — direct invocation

### Runner

- [x] Retry runner — honour `retry:` + `retry_delay:` fields, log each attempt
- [x] Watch mode (`cx run --watch`)

### Reporter

- [x] JUnit XML reporter (`--reporter junit`)
- [x] HTML reporter (`--reporter html`) — self-contained single-file report

### Commands

- [ ] `cx explain --ai` — Claude API powered explanation (`CROSSCHECK_AI_KEY`)

### Config

- [x] `.crosscheck.yaml` global config file (`reporter`, `timeout`, `insecure`, `env-file`)

### Mock server

- [x] Outbound call capture — assert calls made to external services

---

## Phase 3

### Documentation

- [x] Getting started guide (install, first test file, run)
- [x] CLI reference — all commands and flags
- [x] YAML schema reference — every top-level key, block, and field with examples
- [x] Adapter guides — postgres, redis, mysql, mongodb, sqs, sns, s3, dynamodb, lambda, mock
- [x] Auth guide — `type: static` and `type: login` patterns
- [x] Variable & interpolation reference — `{{ VAR }}`, capture, chaining
- [x] Environment priority reference — CLI > shell > `.env` > YAML defaults
- [x] Global config reference — `.crosscheck.yaml` options
- [x] Reporter guide — pretty, json, junit, html, `--output-file`
- [x] Watch mode guide
- [x] CI/CD integration guide — GitHub Actions example workflow
- [x] FAQ / troubleshooting

---

## Phase 4

### Example project — Node.js Express + Redis + MariaDB

- [x] `examples/users-api/` — Express app:
  - `POST /users` — insert user row into MariaDB, cache in Redis, fire webhook
  - `GET /users/:id` — return user (read from MariaDB)
- [x] `examples/users-api/docker-compose.yml` — services: app, Redis, MariaDB
- [x] `examples/users-api/db/init.sql` — creates `users` table on MariaDB startup
- [x] `examples/users-api/tests/users.cx.yaml` — full crosscheck test suite:
  - Create user → assert HTTP 201 + capture `userId`
  - Fetch user → assert HTTP 200 + body fields
  - MariaDB assertion — `users` row written with correct fields (`adapter: mysql`)
  - Redis assertion — `user:<id>` key cached with correct name (`adapter: redis`)
  - Mock server — assert webhook fired with correct payload (`adapter: mock`)
- [x] `examples/users-api/README.md` — walkthrough: `docker compose up`, `cx run ./tests/`
- [x] GitHub Actions workflow — `docker compose up -d`, wait for health checks, `cx run`, `docker compose down`

---

## Phase 5 — Example: Multi-Service Order Pipeline (MariaDB + Redis + RabbitMQ)

> **Scenario:** An e-commerce order pipeline with three Node.js services communicating async via RabbitMQ.
> Demonstrates cross-service assertions, `wait_for` polling, variable chaining across services, and mock capture.

### New crosscheck adapter

- [ ] RabbitMQ adapter (`rabbitmq/amqp091-go`) — connect to broker, peek/consume message from named queue, assert payload fields
  - Named as `adapter: rabbitmq` in test files
  - Config fields: `url`, `queue`, `routing_key`, `exchange`
  - `wait_for` polling supported (messages may arrive asynchronously)

### Services (`examples/order-pipeline/`)

- [x] **order-api** (`src/order-api/`) — `POST /orders`, `GET /orders/:id`, `GET /products/:id`
  - On `POST /orders`: validates stock, inserts row in MariaDB `orders`, decrements `products.stock`, publishes `order.placed` event to RabbitMQ `orders` exchange
  - Returns `{ orderId, productId, status: "placed" }`
- [x] **inventory-service** (`src/inventory-service/`) — subscribes to `orders` queue
  - Consumes `order.placed` events
  - Rebuilds and caches current stock in Redis key `stock:<productId>`
  - Fires a `POST /low-stock` webhook when stock drops below threshold (capturable via mock server)
- [x] **notification-service** (`src/notification-service/`) — subscribes to `orders` queue
  - Consumes `order.placed` events
  - Fires a `POST /notify` webhook with `{ event, orderId, productId }` (capturable via mock server)

### Infrastructure

- [x] `examples/order-pipeline/docker-compose.yml`
  - Services: `mariadb`, `redis`, `rabbitmq` (with management UI on `:15672`), `order-api`, `inventory-service`, `notification-service`
  - Health checks on all infrastructure services before app containers start
- [x] `examples/order-pipeline/db/init.sql` — creates `orders` and `products` tables, seeds product rows with initial stock
- [x] `examples/order-pipeline/rabbitmq/definitions.json` — pre-declares exchange, queues, bindings (incl. `assert-orders` mirror queue for test assertions)

### crosscheck test suites

- [x] `examples/order-pipeline/tests/place-order.cx.yaml`
  - `setup`: truncate `orders` table, reset product stock to 2, flush Redis, purge assert queue
  - **Step 1 — Place order**: `POST /orders` → assert HTTP 201, capture `orderId` + `productId`
  - **Step 2 — Order in DB**: MariaDB — `orders` row exists with `status: placed` (`adapter: mysql`)
  - **Step 3 — Message on queue**: RabbitMQ — `order.placed` event in `assert-orders` queue (`adapter: rabbitmq`, `wait_for: { timeout: 5s, interval: 200ms }`)
  - **Step 4 — Stock decremented**: wait_for MariaDB `products.stock` to decrease by 1 (`adapter: mysql`, `wait_for: { timeout: 10s, interval: 500ms }`)
  - **Step 5 — Redis cache updated**: `stock:<productId>` key reflects new stock level (`adapter: redis`)
  - **Step 6 — Low-stock webhook**: mock server received `POST /low-stock` with matching `productId` (`adapter: mock`)
  - **Step 7 — Notification webhook**: mock server received `POST /notify` with `orderId` + `event: order.placed` (`adapter: mock`)
- [x] `examples/order-pipeline/tests/out-of-stock.cx.yaml`
  - **Step 1 — Drain stock**: set product stock to 0 via DB setup hook
  - **Step 2 — Reject order**: `POST /orders` → assert HTTP 409 + error body
  - **Step 3 — No DB row**: MariaDB — confirm no `orders` row was inserted

### Docs

- [x] `examples/order-pipeline/README.md` — architecture diagram (ASCII), prerequisites, `docker compose up --build`, `cx run ./tests/`, tear-down

### CI

- [ ] GitHub Actions job — `docker compose up -d`, wait for all health checks, `cx run examples/order-pipeline/tests/`, `docker compose down`

---

## Phase 6

### VS Code Extension

- [ ] TextMate grammar — `{{ VAR }}`, `:namedParam`, top-level keys
- [ ] Schema autocomplete via `yaml-language-server` + `crosscheck.schema.json`
- [ ] CodeLens "▶ Run test" per test block
- [ ] VS Code native Test Explorer integration
- [ ] Inline failure decorations (gutter + hover)
- [ ] Environment switcher in status bar
- [ ] `cx explain` side panel
- [ ] DB query preview on hover

### Other

- [ ] Test generation from OpenAPI spec
- [ ] Parallel test execution
- [ ] OAuth2 support (client credentials, PKCE)

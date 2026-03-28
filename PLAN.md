# E2E API Test Tool

> A config-driven CLI that tests backend APIs end-to-end: HTTP call → response assertion → DB/service state assertion.
>
> **Dual purpose:** Portfolio project built in Go (language learning) + real work tool used in production CI pipelines. Quality and reliability are non-negotiable — this runs on real services.

## Problem

Existing tools only test one layer at a time:

- **Hurl / curl** — HTTP only, no DB assertions
- **Postman / Insomnia** — GUI, no native DB assertions, not CI-native
- **Supertest** — in-process, Node.js only, no DB assertions
- **k6** — load testing focus, not assertion-driven
- **Playwright** — browser-focused, awkward for pure API + DB flows

None of them answer: _"Did the API call actually write the right data?"_

## Core Value

Run a test that:

1. Calls `POST /orders`
2. Asserts the response is `201` with the right shape
3. Asserts a row was inserted in `orders` table with the correct values
4. Asserts a Redis key was set
5. Asserts a message was published to SQS
6. Asserts an item was written to DynamoDB
7. Asserts an outbound webhook was fired

All from one test definition file, from the CLI, in CI.

---

## Name

`crosscheck` — alias `cx`

---

## Language

**Go** — single binary distribution, fast startup, no runtime dependency, natural fit for CLI tooling.

### Key packages

| Purpose                | Package                                                                                     |
| ---------------------- | ------------------------------------------------------------------------------------------- |
| CLI framework          | [`github.com/spf13/cobra`](https://github.com/spf13/cobra)                                  |
| YAML parsing           | [`gopkg.in/yaml.v3`](https://pkg.go.dev/gopkg.in/yaml.v3)                                   |
| HTTP client            | stdlib `net/http`                                                                           |
| JSON assertions        | [`github.com/tidwall/gjson`](https://github.com/tidwall/gjson) — JSONPath-style reads       |
| Postgres               | [`github.com/jackc/pgx/v5`](https://github.com/jackc/pgx)                                   |
| MySQL                  | `database/sql` + [`github.com/go-sql-driver/mysql`](https://github.com/go-sql-driver/mysql) |
| MongoDB                | [`go.mongodb.org/mongo-driver`](https://www.mongodb.com/docs/drivers/go/current/)           |
| Redis                  | [`github.com/redis/go-redis/v9`](https://github.com/redis/go-redis)                         |
| Colored output         | [`github.com/fatih/color`](https://github.com/fatih/color)                                  |
| Template interpolation | lightweight `strings.Replacer` / regex — `{{ VAR }}` syntax (not `text/template`)           |
| `.env` file loading    | [`github.com/joho/godotenv`](https://github.com/joho/godotenv)                              |
| Release automation     | [`goreleaser`](https://goreleaser.com) — cross-compile, GitHub Releases, Homebrew formula   |
| AWS SDK                | [`github.com/aws/aws-sdk-go-v2`](https://github.com/aws/aws-sdk-go-v2) — SQS, SNS, S3, DynamoDB, Lambda |

---

## Distribution

Three install methods, all automated by `goreleaser` on every git tag:

| Method | Command | Requires |
|--------|---------|----------|
| **Script** | `curl -sSL .../install.sh \| sh` | Nothing |
| **Homebrew** | `brew install lozymon/tap/crosscheck` | Homebrew |
| **Go install** | `go install github.com/lozymon/crosscheck@latest` | Go toolchain |

`goreleaser` handles cross-compilation (Linux, macOS, Windows — amd64 + arm64), checksums, GitHub Release creation, and Homebrew formula generation from a single `.goreleaser.yaml` config file.

---

## Architecture

```
crosscheck/
├── main.go
├── .goreleaser.yaml  # Cross-compile + GitHub Releases + Homebrew formula
├── cmd/              # Cobra commands (run, validate, init, explain)
├── runner/           # Test file loader, execution engine, step chaining
├── httpclient/       # net/http wrapper, request builder
├── adapters/         # DB + service adapters (Go interface)
│   ├── postgres/     # pgx/v5
│   ├── mysql/        # go-sql-driver
│   ├── mongodb/      # mongo-driver
│   ├── redis/        # go-redis/v9
│   ├── sqs/          # aws-sdk-go-v2
│   ├── sns/          # aws-sdk-go-v2
│   ├── s3/           # aws-sdk-go-v2
│   ├── dynamodb/     # aws-sdk-go-v2
│   ├── lambda/       # aws-sdk-go-v2
│   └── mockserver/   # Capture outbound calls
├── assert/           # Assertion engine (gjson, regex, deep equal)
├── reporter/         # Output formatters (pretty/color, JSON, JUnit XML)
├── config/           # YAML schema structs + parser
├── interpolate/      # text/template-based variable interpolation
└── env/              # .env loading (godotenv), priority merge (CLI > shell > .env > yaml)
```

---

## Test Definition Format

### Environment variables

Variables are resolved in this priority order (highest wins):

1. `--env KEY=VALUE` CLI flags
2. Shell environment (`export BASE_URL=...`)
3. `.env` file in the working directory
4. `env:` block in the YAML test file (defaults / documentation)

```bash
# .env  (never commit this)
BASE_URL=http://localhost:3000
DB_URL=postgres://user:pass@localhost/testdb
AUTH_TOKEN=supersecret
```

The `.env` file is loaded automatically if present. A different file can be specified with `--env-file`:

```bash
crosscheck run --env-file .env.staging
```

### YAML (primary)

```yaml
# crosscheck.cx.yaml
version: 1
name: Create order flow
description: 'Verifies that an order is created, persisted to the DB, and cached in Redis'
env:
  # Fallback defaults — override via .env or CLI flags
  base_url: http://localhost:3000
  db: postgres://user:pass@localhost/testdb # never put real creds here

tests:
  - name: Create an order
    description: 'POST /orders should return 201, insert a row in orders, and set a Redis cache key'
    timeout: 5s
    retry: 3
    retry_delay: 1s
    request:
      method: POST
      url: '{{ BASE_URL }}/orders'
      headers:
        Content-Type: application/json
        Authorization: 'Bearer {{ AUTH_TOKEN }}'
      body:
        productId: 'abc-123'
        quantity: 2

    response:
      status: 201
      body:
        id: '{{ capture: orderId }}' # captured into .Vars.orderId for later steps
        status: pending

    database:
      - adapter: postgres
        query: 'SELECT status, quantity FROM orders WHERE id = :orderId'
        params:
          orderId: '{{ orderId }}'
        expect:
          - status: pending
            quantity: 2

    services:
      - adapter: redis
        key: 'order:{{ orderId }}'
        expect:
          status: pending

  - name: Get the created order
    request:
      method: GET
      url: '{{ BASE_URL }}/orders/{{ orderId }}'
    response:
      status: 200
      body:
        id: '{{ orderId }}'
        status: pending
```

### Setup / teardown hooks

For anything YAML can't express (seeding a DB, spinning up a mock, cleaning up state), use shell hooks. These replace the need for a Go DSL.

```yaml
setup:
  - run: 'psql $DB_URL -f ./fixtures/seed.sql'
  - run: 'docker compose up -d mockserver --wait'

teardown:
  - run: 'psql $DB_URL -f ./fixtures/cleanup.sql'
  - run: 'docker compose stop mockserver'
```

`setup` runs once before all tests in the file. `teardown` runs after, even if tests fail.

Per-test hooks are also supported:

```yaml
tests:
  - name: Create an order
    setup:
      - run: 'psql $DB_URL -c "DELETE FROM orders"'
    teardown:
      - run: 'psql $DB_URL -c "DELETE FROM orders"'
    request: ...
```

---

## CLI Interface

Both `crosscheck` and `cx` are valid — `cx` is the short alias.

### File discovery

| Command | Behaviour |
|---------|-----------|
| `cx run` | Recursively finds all `**/*.cx.yaml` in current directory |
| `cx run ./tests/` | Recursively finds all `**/*.cx.yaml` in `./tests/` |
| `cx run ./tests/orders.cx.yaml` | Runs a specific file (any `.yaml` accepted — no suffix restriction on explicit paths) |

`cx init` creates `crosscheck.cx.yaml` in the current directory as the conventional root test file.

```bash
# Run all test files in current dir
cx run

# Run a specific file
cx run ./tests/orders.yaml

# Use a specific .env file
cx run --env-file .env.staging

# Override a single variable
cx run --env BASE_URL=http://staging.api.com

# Watch mode
cx run --watch

# Skip TLS verification (staging with self-signed certs)
cx run --insecure

# Filter tests by name pattern
cx run --filter "order*"

# Output formats
cx run --reporter junit > results.xml
cx run --reporter json
cx run --reporter pretty   # default

# Write JSON results to file alongside pretty output
cx run --output-file results.json

# Explain what a test file does in plain English
cx explain ./tests/orders.yaml
```

### Global config (Phase 2)

`.crosscheck.yaml` at project root sets defaults — no flags needed in CI:

```yaml
# .crosscheck.yaml
reporter: pretty
timeout: 10s
insecure: false
env-file: .env
```

Override with `--config ./other.yaml` or any CLI flag.

### Exit codes

| Code | Meaning                                 |
| ---- | --------------------------------------- |
| `0`  | All tests passed                        |
| `1`  | One or more tests failed                |
| `2`  | Config / YAML validation error          |
| `3`  | Connection error (DB, HTTP unreachable) |

---

## AI & Documentation

### JSON Schema for test files

A `crosscheck.schema.json` is published alongside the binary. It enables:

- VS Code autocomplete and inline validation (via `yaml-language-server` comment)
- AI tools that generate or validate test files
- `cx validate` schema enforcement

```yaml
# yaml-language-server: $schema=https://crosscheck.dev/schema.json
name: Create order flow
```

### Structured failure output

Every test failure emits a full JSON object — readable by humans, parseable by AI:

```json
{
  "test": "Create an order",
  "step": "response.body.status",
  "passed": false,
  "expected": "pending",
  "actual": "draft",
  "request": {
    "method": "POST",
    "url": "http://localhost:3000/orders",
    "body": { "productId": "abc-123", "quantity": 2 }
  },
  "response": {
    "status": 201,
    "body": { "id": "ord_123", "status": "draft" }
  }
}
```

### `cx explain` command

Reads a test file and outputs a plain-English summary — useful for onboarding, PR reviews, and documentation generation.

**Static mode (default)** — parses YAML and formats into a readable summary, no dependencies:

```bash
$ cx explain ./tests/orders.cx.yaml

File: orders.cx.yaml
Suite: Create order flow
Description: Verifies that an order is created, persisted to the DB, and cached in Redis

Tests:
  1. Create an order — POST /orders, expects 201, checks orders table + Redis cache key
  2. Get the created order — GET /orders/:id, expects 200 with matching id and status
```

**AI mode (Phase 2)** — pipes the test file to an LLM for a richer explanation that captures intent, not just structure:

```bash
$ cx explain --ai ./tests/orders.cx.yaml

This test verifies the full order creation flow. It sends a POST request
to create an order, confirms the API returns the new order ID, then
cross-checks that the database actually persisted the record with the
correct status — catching cases where the API returns success but the
write silently fails. It also verifies the Redis cache is populated,
which downstream services depend on for read performance.
```

Requires `CROSSCHECK_AI_KEY` in `.env`. Claude API is the default provider.

### `description:` field

Both suites and individual tests support a `description:` field. It appears in reports and structured output, giving AI tools context about _why_ a test exists — not just what it does.

---

## DB Adapter Interface

```go
type Adapter interface {
    Name() string
    Connect(ctx context.Context, config AdapterConfig) error
    Query(ctx context.Context, query string, params ...any) ([]map[string]any, error)
    Disconnect(ctx context.Context) error
}
```

Built-in adapters:

- `postgres` — `pgx/v5`
- `mysql` — `database/sql` + `go-sql-driver`
- `mongodb` — `mongo-driver`
- `redis` — `go-redis/v9`

---

## Assertion Engine

Support for:

- Exact value: `status: pending`
- Partial match: `body contains { id: "..." }`
- JSONPath: `$.items[0].name == "foo"`
- Regex: `email: /^.+@.+\..+$/`
- Capture: `id: "{{ capture: orderId }}"` — stores for use in later steps

### `wait_for` — async / eventual consistency polling

For flows where an API call triggers async work (SQS → consumer → DB write), `wait_for` polls a DB assertion until it matches or times out. This is different from `retry:` — it polls only the assertion, not the HTTP request.

```yaml
database:
  - adapter: postgres
    query: "SELECT capacity FROM stock WHERE event_id = :eventId"
    params:
      eventId: "{{ eventId }}"
    wait_for:
      timeout: 10s      # give up after 10s
      interval: 500ms   # poll every 500ms
    expect:
      - capacity: 9
```

Without `wait_for`, async flows require artificial `sleep` in teardown hooks or fail intermittently. Critical for microservice + queue architectures.

---

## MVP Scope

Goal: working end-to-end in the simplest case.

- [ ] YAML test file parser with `version:`, `name:`, and `description:` fields (missing version assumes `1` with a warning)
- [ ] JSON Schema (`crosscheck.schema.json`) — single source of truth for all fields
- [ ] `.env` file loading + variable interpolation (`{{ VAR }}` flat syntax for both env and captured vars)
- [ ] `auth:` block (static token + custom login endpoint)
- [ ] HTTP client with chaining (variable capture)
- [ ] HTTP response assertions (status, headers, body)
- [ ] `wait_for` polling on DB assertions (`timeout` + `interval`) — for async queue-driven flows
- [ ] `setup` / `teardown` shell hooks (file-level and per-test)
- [ ] Postgres adapter (named params `:varName` → `$1` rewriting, flat `params:` map)
- [ ] `timeout:` per test
- [ ] `--insecure` CLI flag — skip TLS verification (for staging environments with self-signed certs)
- [ ] `retry:` and `retry_delay:` fields in schema (runner implementation in Phase 2)
- [ ] Pretty CLI reporter
- [ ] Structured JSON failure output (request, response, expected, actual)
- [ ] Machine-readable exit codes (`0` pass / `1` fail / `2` config error / `3` connection error)
- [ ] `cx run` command + `--filter` flag + `--output-file` flag
- [ ] `cx validate` command — dry-run YAML schema check, no requests fired (CI-friendly)
- [ ] `cx init` command — scaffolds a heavily commented `crosscheck.cx.yaml` with `yaml-language-server` schema hint
- [ ] `cx explain` command — plain-English summary of a test file
- [ ] `goreleaser` config — GitHub Releases with binaries for Linux/macOS/Windows (amd64 + arm64)
- [ ] Homebrew tap
- [ ] Install script (`install.sh`)

## Phase 2

- [ ] Redis adapter
- [ ] MongoDB adapter
- [ ] MySQL adapter
- [ ] Watch mode
- [ ] JUnit XML reporter
- [ ] Mock server capture (outbound calls)
- [ ] `cx explain --ai` — LLM-powered explanation via Claude API (`CROSSCHECK_AI_KEY`)
- [ ] Retry runner implementation — honour `retry:` + `retry_delay:` fields, log each attempt
- [ ] SQS adapter — assert messages published to queues after API calls
- [ ] SNS adapter — assert notifications published to topics
- [ ] S3 adapter — assert files written to buckets
- [ ] DynamoDB adapter — assert items written/updated
- [ ] Lambda adapter — direct invocation (not via API Gateway)
- [ ] Kafka / MSK adapter
- [ ] `.crosscheck.yaml` global config file — project-wide defaults (`reporter`, `timeout`, `insecure`, `env-file`); priority: CLI flags > config file > built-in defaults; discovered automatically at project root, overridable with `--config`

## Phase 3

### VS Code Extension

A dedicated extension built on VS Code's native test API and TextMate grammars.

**Syntax highlighting** (TextMate grammar for `.yaml` files):

- `{{ VAR }}` and `{{ capture: x }}` — highlighted as template expressions
- `:namedParam` in SQL strings — visually distinct from surrounding SQL
- Top-level keys (`request:`, `response:`, `database:`, `services:`, `setup:`, `teardown:`) — distinct colors
- `auth:` block — visually separated from test steps

**Features:**

| Feature                        | Description                                                                                       |
| ------------------------------ | ------------------------------------------------------------------------------------------------- |
| **CodeLens**                   | "▶ Run test" button above each test block — runs `cx run --filter` for that test only             |
| **Test Explorer**              | Sidebar panel using VS Code's native test API — shows all suites/tests with pass/fail/skip status |
| **Inline failure decorations** | Red gutter icon + hover tooltip showing expected vs actual on the failing assertion line          |
| **Environment switcher**       | Status bar dropdown to switch between `.env`, `.env.staging`, `.env.prod`                         |
| **`cx explain` side panel**    | Opens a side panel with plain-English summary of the current test file                            |
| **DB query preview**           | Hover over a `query:` block to see rendered SQL with params substituted                           |
| **Schema autocomplete**        | Driven by `crosscheck.schema.json` — field suggestions, type hints, required field warnings       |

**Checklist:**

- [ ] TextMate grammar for crosscheck YAML syntax
- [ ] Schema autocomplete via `yaml-language-server` + `crosscheck.schema.json`
- [ ] CodeLens "▶ Run test" per test block
- [ ] VS Code native Test Explorer integration
- [ ] Inline failure decorations (gutter + hover)
- [ ] Environment switcher in status bar
- [ ] `cx explain` side panel
- [ ] DB query preview on hover

### Other Phase 3

- [ ] GitHub Actions integration
- [ ] Test generation from OpenAPI spec
- [ ] Parallel test execution
- [ ] OAuth2 support (client credentials, PKCE)

---

## Open Questions

_(none — all questions resolved)_

## Decisions Made

- **YAML only, no Go DSL** — shell `setup`/`teardown` hooks cover complex flow needs without a second format to maintain
- **Auth: login endpoint + static token for MVP, no OAuth** — OAuth adds significant complexity and test envs should have a local auth endpoint; OAuth goes in Phase 3 if there's demand
- **No auto-rollback** — wrapping tests in transactions changes app behavior and doesn't work for non-SQL adapters; explicit `teardown` hooks are the isolation strategy
- **Name: `crosscheck` + alias `cx`** — clear on Go/GitHub, no dominant CLI conflicts; `cx` chosen over `cc` to avoid C compiler clash
- **Flat `{{ VAR }}` interpolation, not `text/template`** — simpler syntax for test authors, familiar to Postman/Hurl users; env vars and captured vars share the same namespace
- **Named params in DB queries (`:varName` syntax)** — adapter rewrites `:orderId` → `$1`/`?` per dialect before executing; `params:` is a flat map; no string interpolation in SQL
- **`validate`, `init`, and `explain` in MVP** — `validate` is essential for CI; `init` lowers the barrier to getting started; `explain` serves both humans and AI
- **`wait_for` polling in MVP** — required for microservice + queue flows where DB writes are async; polls assertion only (not HTTP request); without it async flows need `sleep` hacks
- **`cx explain` static in MVP, `--ai` flag in Phase 2** — static parses YAML into a formatted summary (no deps); AI mode pipes to Claude API for intent-aware explanations; requires `CROSSCHECK_AI_KEY`
- **`.crosscheck.yaml` global config in Phase 2** — minimal defaults only (`reporter`, `timeout`, `insecure`, `env-file`); CLI flags > config > built-in defaults; not in MVP since CLI flags are sufficient to get started
- **`--insecure` flag in MVP** — staging environments commonly have self-signed certs; environment concern not per-test; also settable via `.crosscheck.yaml` in Phase 2
- **`retry:` + `retry_delay:` in schema now, runner in Phase 2** — fields added to schema immediately so existing test files don't need updating; useful for eventually-consistent systems; avoids artificial sleeps in teardown hooks
- **JSON Schema as single source of truth** — drives `cx validate`, VS Code autocomplete, and AI tooling from one file
- **`description:` field on suites and tests** — gives AI context about intent, not just structure; shows up in reports and `cx explain` output
- **Structured JSON failure output** — full request/response/expected/actual in every failure; machine-parseable for AI agents and CI tools
- **Machine-readable exit codes** — `0/1/2/3` so CI scripts and AI agents can branch without parsing output text
- **Distribution: GitHub Releases + Homebrew + `go install`** — `goreleaser` automates all three from one config; GitHub Releases first (no runtime needed), Homebrew second, `go install` free automatically
- **Test file naming: `*.cx.yaml`** — consistent with `cx` alias, short, `.yaml` suffix keeps editor highlighting; auto-discovery uses `**/*.cx.yaml`; explicit paths accept any `.yaml`; `cx init` creates `crosscheck.cx.yaml`
- **`version: 1` in all test files** — missing version assumes `1` with a warning; enables clean breaking changes in future schema versions without legacy compatibility hacks

---

## Comparable Tools (research)

- [Hurl](https://hurl.dev) — closest in spirit, HTTP + assertions, but no DB layer
- [Karate DSL](https://github.com/karatelabs/karate) — Java-based, does API + DB, but heavy
- [Tavern](https://taverntesting.github.io) — Python, YAML-driven API tests, no DB
- [Pact](https://pact.io) — contract testing, different use case

The gap: a **lightweight, language-agnostic, DB-aware** CLI test runner.

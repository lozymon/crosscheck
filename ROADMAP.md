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

## Phase 5

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

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
- [ ] File-level `setup` runs before all tests
- [ ] File-level `teardown` runs after all tests (even on failure)
- [ ] Per-test `setup` / `teardown`
- [ ] Shell command execution with env vars available

### Variable chaining
- [ ] Capture vars from response body (`{{ capture: varName }}`)
- [ ] Captured vars merged into interpolation namespace
- [ ] Vars available across all subsequent steps in the file

### Reporter
- [ ] Pretty CLI output (colored pass/fail per test)
- [ ] Structured JSON failure output (request, response, expected, actual)
- [ ] `--output-file` writes JSON results alongside pretty output
- [ ] Exit codes: `0` pass / `1` fail / `2` config error / `3` connection error

### File discovery
- [ ] `cx run` with no args finds `**/*.cx.yaml` recursively
- [ ] `cx run ./path/` finds `**/*.cx.yaml` in given directory
- [ ] `cx run ./file.yaml` runs specific file (any `.yaml` accepted)
- [ ] `--filter` pattern matching against test names

### Commands implementation
- [ ] `cx validate` — parse + schema check, no requests fired
- [ ] `cx init` — scaffold commented `crosscheck.cx.yaml` with schema hint
- [ ] `cx explain` — static plain-English summary of a test file

### Schema
- [ ] `crosscheck.schema.json` published alongside binary

### Distribution
- [ ] `.goreleaser.yaml` config
- [ ] GitHub Actions release workflow (triggers on git tag)
- [ ] Homebrew tap (`lozymon/homebrew-tap`)
- [ ] `install.sh` curl script

---

## Phase 2

### Adapters
- [ ] Redis adapter (`go-redis/v9`)
- [ ] MySQL adapter (`go-sql-driver`)
- [ ] MongoDB adapter (`mongo-driver`)
- [ ] SQS adapter (`aws-sdk-go-v2`) — assert messages published to queue
- [ ] SNS adapter (`aws-sdk-go-v2`) — assert notifications published
- [ ] S3 adapter (`aws-sdk-go-v2`) — assert objects written to bucket
- [ ] DynamoDB adapter (`aws-sdk-go-v2`) — assert items written/updated
- [ ] Lambda adapter (`aws-sdk-go-v2`) — direct invocation

### Runner
- [ ] Retry runner — honour `retry:` + `retry_delay:` fields, log each attempt
- [ ] Watch mode (`cx run --watch`)

### Reporter
- [ ] JUnit XML reporter (`--reporter junit`)

### Commands
- [ ] `cx explain --ai` — Claude API powered explanation (`CROSSCHECK_AI_KEY`)

### Config
- [ ] `.crosscheck.yaml` global config file (`reporter`, `timeout`, `insecure`, `env-file`)

### Mock server
- [ ] Outbound call capture — assert calls made to external services

---

## Phase 3

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
- [ ] GitHub Actions integration docs + example workflow
- [ ] Test generation from OpenAPI spec
- [ ] Parallel test execution
- [ ] OAuth2 support (client credentials, PKCE)

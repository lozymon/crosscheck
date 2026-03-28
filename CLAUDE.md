# CLAUDE.md

This file provides guidance to Claude Code (claude.ai/code) when working with code in this repository.

## What This Project Is

`crosscheck` (alias `cx`) is a Go CLI tool for end-to-end API testing that validates HTTP responses, database state, and cloud services from a single YAML test file. It's in early MVP stage — the architecture and YAML schema are well-defined, but most execution logic is still stubs.

## Commands

```bash
# Build
go build -o crosscheck ./main.go

# Run all tests
go test ./...

# Run a specific package
go test ./config/...

# Run with verbose output
go test -v ./...

# Install locally
go install .
```

No Makefile yet — use `go` commands directly.

## Architecture

The project follows a pipeline: **CLI → Config Parser → Env Manager → Interpolator → Runner → Adapters → Assertions → Reporter**

```
cmd/           # Cobra CLI commands (run, validate, init, explain)
config/        # YAML schema structs (types.go) + parser (parser.go)
env/           # Variable priority merging: CLI > shell > .env > YAML defaults
interpolate/   # {{ VAR }} expansion across strings and maps
httpclient/    # HTTP request builder (empty — Phase 1 target)
runner/        # Test execution orchestrator (empty — Phase 1 target)
assert/        # Assertion logic (empty — Phase 1 target)
reporter/      # Output formatters: pretty, JSON, JUnit (empty — Phase 1 target)
adapters/      # Per-service query executors (stub directories for postgres, redis, etc.)
examples/      # Example .cx.yaml test files
```

**Implementation status:**
- **Done:** `config/`, `env/`, `interpolate/`, `cmd/` (CLI structure with flags, commands print stubs)
- **Empty — Phase 1:** `httpclient/`, `runner/`, `assert/`, `reporter/`
- **Empty — Phase 2:** adapter packages (postgres, redis, mysql, mongodb, dynamodb, s3, sqs, sns)

## Key Design Decisions

**Variable interpolation:** Uses simple regex (`{{ VAR }}`) instead of `text/template` — intentional to keep YAML readable and avoid template syntax conflicts. Lives in `interpolate/interpolate.go`.

**Environment priority** (highest to lowest): `--env` CLI flags → shell exports → `.env` file → YAML `env:` block defaults. Implemented in `env/env.go`.

**Named DB params:** Postgres queries use `:varName` syntax (e.g., `WHERE id = :orderId`) which the adapter converts to `$1` placeholders. Keeps YAML readable.

**Test file discovery:** `cx run` finds `**/*.cx.yaml` recursively. Explicit file paths accept any `.yaml` suffix.

## YAML Test Format

Test files follow this structure (see `examples/subscription-flow.cx.yaml` for a full example):

```yaml
version: 1
name: Suite name
env:
  BASE_URL: http://localhost:3000  # fallback defaults

auth:
  type: login  # or "static"
  request:
    method: POST
    url: "{{ AUTH_SERVICE }}/auth/login"
    body: { email: "...", password: "..." }
  capture:
    token: "$.accessToken"
  inject:
    header: Authorization
    format: "Bearer {{ token }}"

setup:
  - run: "psql $DB_URL -f ./fixtures/seed.sql"

tests:
  - name: Test name
    request:
      method: POST
      url: "{{ BASE_URL }}/orders"
      body: { productId: "abc" }
    response:
      status: 201
      body:
        id: "{{ capture: orderId }}"  # saves $.id as orderId
    database:
      - adapter: postgres
        query: "SELECT status FROM orders WHERE id = :orderId"
        params: { orderId: "{{ orderId }}" }
        wait_for: { timeout: 10s, interval: 500ms }
        expect:
          - status: pending
    services:
      - adapter: redis
        key: "order:{{ orderId }}"
        expect: { status: pending }

teardown:
  - run: "psql $DB_URL -f ./fixtures/cleanup.sql"
```

## CLI Interface

```bash
cx run                           # Find all **/*.cx.yaml recursively
cx run ./tests/orders.yaml       # Run specific file
cx run --env-file .env.staging
cx run --env BASE_URL=http://staging.com
cx run --filter "order*"
cx run --reporter json           # pretty (default), json, junit
cx run --output-file results.json
cx validate                      # Schema check, no HTTP/DB calls
cx init                          # Scaffold crosscheck.cx.yaml template
cx explain ./tests/orders.yaml   # Plain-English summary
```

**Exit codes:** 0 = all passed, 1 = test failures, 2 = YAML validation error, 3 = connection error.

## Module

```
github.com/lozymon/crosscheck
```

Key dependencies: `spf13/cobra`, `gopkg.in/yaml.v3`, `joho/godotenv`, `fatih/color`.

## Roadmap Context

- **PLAN.md** — Full design document with architecture decisions and YAML schema rationale
- **ROADMAP.md** — Phased implementation checklist (MVP → Phase 2 adapters → Phase 3 VS Code extension)

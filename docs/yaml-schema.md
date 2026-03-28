# YAML Schema Reference

Every `*.cx.yaml` test file follows this structure. All fields are optional unless marked **required**.

```yaml
# yaml-language-server: $schema=./crosscheck.schema.json
version: 1          # required
name: my suite
description: optional description

env: ...            # variable defaults
auth: ...           # authentication
mock: ...           # mock server
setup: ...          # file-level setup hooks
teardown: ...       # file-level teardown hooks
tests: ...          # required — test list
```

---

## Top-level fields

| Field | Type | Description |
|---|---|---|
| `version` | int | **Required.** Must be `1`. |
| `name` | string | Suite name shown in reporter output. |
| `description` | string | Free-text description. |
| `env` | map | Variable defaults (lowest priority). See [Environment](environment.md). |
| `auth` | object | Authentication config run once before all tests. See [Auth](auth.md). |
| `mock` | object | Local mock HTTP server for outbound call capture. |
| `setup` | list | Shell commands run before all tests. |
| `teardown` | list | Shell commands run after all tests (always runs). |
| `tests` | list | **Required.** List of test objects. |

---

## `env`

Key/value map of variable defaults. These are the lowest-priority source — overridden by `.env` file, shell, and `--env` flags.

```yaml
env:
  BASE_URL: http://localhost:3000
  API_KEY: dev-key
```

---

## `mock`

Starts a local HTTP server that captures all incoming requests. The server URL is injected as `MOCK_URL` so you can pass it to your application.

```yaml
mock:
  port: 9099   # 0 or omit = auto-assign a free port
```

Assert captured calls with `adapter: mock` in the `services` block.

---

## `setup` / `teardown`

Lists of shell commands. File-level setup runs before all tests; teardown always runs even if tests fail.

```yaml
setup:
  - run: psql $DB_URL -f fixtures/seed.sql
  - run: docker compose up -d --wait

teardown:
  - run: psql $DB_URL -f fixtures/cleanup.sql
```

---

## `tests`

List of test objects.

```yaml
tests:
  - name: create order          # required
    description: optional
    timeout: 10s                # per-test timeout
    retry: 2                    # retry up to 2 extra times on failure
    retry_delay: 500ms          # wait between retries
    setup: [...]                # per-test setup hooks
    teardown: [...]             # per-test teardown hooks
    request: ...                # HTTP request
    response: ...               # HTTP response assertions
    database: [...]             # database assertions
    services: [...]             # service assertions
```

### `request`

```yaml
request:
  method: POST          # required
  url: "{{ BASE_URL }}/orders"   # required, supports {{ VAR }}
  headers:
    Content-Type: application/json
    X-Api-Key: "{{ API_KEY }}"
  body:
    productId: abc
    quantity: 1
```

`body` can be a map (serialised as JSON) or a plain string.

### `response`

```yaml
response:
  status: 201
  headers:
    Content-Type: application/json
  body:
    id: "{{ capture: orderId }}"   # captures $.id into orderId
    status: pending
```

All fields are optional. Omit `status` to skip status assertion. Body values support partial matching — only the keys listed are checked.

### `database`

List of database assertion blocks. See [Adapters](adapters.md) for per-adapter details.

```yaml
database:
  - adapter: postgres   # postgres | mysql | mongodb
    query: "SELECT status FROM orders WHERE id = :orderId"
    params:
      orderId: "{{ orderId }}"
    wait_for:
      timeout: 10s
      interval: 500ms
    expect:
      - status: pending
```

### `services`

List of service assertion blocks. See [Adapters](adapters.md) for per-adapter details.

```yaml
services:
  - adapter: redis
    key: "order:{{ orderId }}"
    expect:
      status: pending

  - adapter: sqs
    queue: https://sqs.us-east-1.amazonaws.com/123/my-queue
    wait_for: { timeout: 10s, interval: 1s }
    expect:
      eventType: order.created

  - adapter: mock
    path: /webhook
    method: POST
    wait_for: { timeout: 5s, interval: 200ms }
    expect:
      event: order.created
```

### `wait_for`

Available on `database` and `services` blocks that support async flows.

```yaml
wait_for:
  timeout: 10s     # how long to keep polling
  interval: 500ms  # how long to wait between attempts
```

Polling stops as soon as the assertion passes or the timeout elapses.

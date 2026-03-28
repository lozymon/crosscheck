# Variables & Interpolation

crosscheck uses `{{ VAR }}` syntax to inject variable values into any string field in a test file.

---

## Basic interpolation

```yaml
env:
  BASE_URL: http://localhost:3000

tests:
  - name: get user
    request:
      method: GET
      url: '{{ BASE_URL }}/users/123'
```

`{{ VAR }}` is replaced with the value of `VAR` from the current variable namespace. Unknown variables are replaced with an empty string.

Interpolation applies to:

- `request.url`
- `request.headers` (values)
- `request.body` (string values, recursively)
- `response.body` (string values)
- `database.query`
- `database.params` (values)
- `database.expect` (values)
- `services.*` fields (`key`, `queue`, `bucket`, `table`, `path`, `method`, `expect` values, `payload` values)
- `auth.request.*`
- `auth.inject.format`
- `setup` / `teardown` hook commands

---

## Capture

Capture extracts a value from the HTTP response body and saves it as a variable for subsequent tests.

```yaml
response:
  body:
    id: '{{ capture: orderId }}'
```

This evaluates `$.id` against the response JSON and stores the result as `orderId`.

The capture syntax `{{ capture: varName }}` is shorthand — the JSONPath expression is always `$.` + the expected value key path.

Captured vars are merged into the variable namespace and available in all subsequent steps of the same test and all later tests in the file.

```yaml
tests:
  - name: create order
    request:
      method: POST
      url: '{{ BASE_URL }}/orders'
      body: { productId: abc }
    response:
      status: 201
      body:
        id: '{{ capture: orderId }}' # saves $.id → orderId

  - name: fetch order
    request:
      method: GET
      url: '{{ BASE_URL }}/orders/{{ orderId }}' # uses captured value
    response:
      status: 200
```

---

## Auth captures

Variables captured during auth (`auth.capture`) are also available in all tests:

```yaml
auth:
  type: login
  capture:
    token: '$.accessToken'
    userId: '$.user.id'

tests:
  - name: get own profile
    request:
      method: GET
      url: '{{ BASE_URL }}/users/{{ userId }}' # userId captured from login
```

---

## MOCK_URL

When a `mock:` block is present in the test file, crosscheck automatically injects `MOCK_URL` into the variable namespace with the mock server's base URL.

```yaml
mock:
  port: 9099

env:
  # Pass the mock server URL to your app via an environment variable
  # (typically done outside the YAML via docker-compose or shell)
  WEBHOOK_URL: 'http://host.docker.internal:9099/'
```

---

## Variable precedence

See [Environment](environment.md) for the full priority order. In short, variables set at runtime (CLI, shell) always win over file-level defaults.

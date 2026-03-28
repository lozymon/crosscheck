# Auth Guide

crosscheck supports authentication that runs **once before all tests**. The resulting token is injected into every subsequent request automatically.

---

## `type: static`

Reads a token directly from an environment variable and injects it as a header.

```yaml
auth:
  type: static
  inject:
    header: Authorization
    format: "Bearer {{ API_KEY }}"
```

- `{{ API_KEY }}` is resolved from your environment (shell, `.env` file, or `--env` flag).
- The formatted value is set on every test request as the specified header.

---

## `type: login`

POSTs to a login endpoint, captures the token from the response, and injects it into all subsequent requests.

```yaml
auth:
  type: login
  request:
    method: POST
    url: "{{ AUTH_SERVICE }}/auth/login"
    headers:
      Content-Type: application/json
    body:
      email: "{{ USER_EMAIL }}"
      password: "{{ USER_PASSWORD }}"
  capture:
    token: "$.accessToken"     # JSONPath into the login response body
  inject:
    header: Authorization
    format: "Bearer {{ token }}"
```

- `capture` maps variable names to JSONPath expressions evaluated against the login response body.
- Captured variables are available in `inject.format` and in all test requests.
- The login request is fired exactly once per test file run.

---

## Per-test override

Tests can override the auth header by setting it explicitly in their own `headers` block:

```yaml
tests:
  - name: unauthenticated request
    request:
      method: GET
      url: "{{ BASE_URL }}/public"
      headers:
        Authorization: ""   # clears the injected header for this test
```

---

## Example — API key auth

```yaml
version: 1
name: API key example

auth:
  type: static
  inject:
    header: X-Api-Key
    format: "{{ API_KEY }}"

tests:
  - name: list items
    request:
      method: GET
      url: "{{ BASE_URL }}/items"
    response:
      status: 200
```

## Example — JWT login

```yaml
version: 1
name: JWT example

auth:
  type: login
  request:
    method: POST
    url: "{{ BASE_URL }}/auth/login"
    body:
      email: "{{ ADMIN_EMAIL }}"
      password: "{{ ADMIN_PASSWORD }}"
  capture:
    token: "$.token"
  inject:
    header: Authorization
    format: "Bearer {{ token }}"

tests:
  - name: get profile
    request:
      method: GET
      url: "{{ BASE_URL }}/me"
    response:
      status: 200
```

# Getting Started

## Install

**curl (Linux/macOS):**

```bash
curl -sSL https://raw.githubusercontent.com/lozymon/crosscheck/main/install.sh | sh
```

**Go install:**

```bash
go install github.com/lozymon/crosscheck@latest
```

**Build from source:**

```bash
git clone https://github.com/lozymon/crosscheck
cd crosscheck
go build -o cx ./main.go
```

---

## Your first test file

Create `hello.cx.yaml`:

```yaml
version: 1
name: hello world

tests:
  - name: get public API
    request:
      method: GET
      url: https://httpbin.org/get
    response:
      status: 200
```

Run it:

```bash
cx run hello.cx.yaml
```

Output:

```
hello world

  ✓  get public API

  1 tests  1 passed  0 failed
```

---

## Add environment variables

```yaml
version: 1
name: my API

env:
  BASE_URL: http://localhost:3000 # fallback default

tests:
  - name: health check
    request:
      method: GET
      url: '{{ BASE_URL }}/health'
    response:
      status: 200
```

Override at runtime:

```bash
cx run --env BASE_URL=http://staging.example.com
```

Or from a `.env` file:

```bash
cx run --env-file .env.staging
```

---

## Capture and chain values

```yaml
tests:
  - name: create item
    request:
      method: POST
      url: '{{ BASE_URL }}/items'
      body: { name: 'widget' }
    response:
      status: 201
      body:
        id: '{{ capture: itemId }}' # saves $.id into itemId

  - name: fetch item
    request:
      method: GET
      url: '{{ BASE_URL }}/items/{{ itemId }}' # uses captured value
    response:
      status: 200
```

---

## Next steps

- [CLI Reference](cli-reference.md) — all commands and flags
- [YAML Schema](yaml-schema.md) — every field explained
- [Variables & Interpolation](variables.md) — `{{ VAR }}`, capture, chaining
- [Adapters](adapters.md) — database and service assertions

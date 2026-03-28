# CLI Reference

## cx run

Run test files.

```
cx run [file or directory] [flags]
```

If no path is given, recursively finds all `*.cx.yaml` files in the current directory.

### Flags

| Flag              | Default            | Description                                      |
| ----------------- | ------------------ | ------------------------------------------------ |
| `--config`        | `.crosscheck.yaml` | Path to global config file                       |
| `--env KEY=VALUE` | —                  | Override an env variable (repeatable)            |
| `--env-file`      | `.env`             | Path to `.env` file                              |
| `--filter`        | —                  | Run only tests whose name matches a glob pattern |
| `--insecure`      | `false`            | Skip TLS certificate verification                |
| `--output-file`   | —                  | Write JSON results to a file                     |
| `--reporter`      | `pretty`           | Output format: `pretty`, `json`, `junit`, `html` |
| `--timeout`       | —                  | Default per-test timeout, e.g. `10s`             |
| `--watch`         | `false`            | Re-run tests on file changes                     |

### Examples

```bash
# Run all *.cx.yaml files recursively
cx run

# Run a specific file
cx run tests/orders.cx.yaml

# Run all files in a directory
cx run tests/

# Override environment
cx run --env BASE_URL=http://staging.example.com

# Use a staging .env file
cx run --env-file .env.staging

# Filter tests by name pattern
cx run --filter "order*"

# Output JUnit XML for CI
cx run --reporter junit --output-file results.xml

# Skip TLS verification for self-signed certs
cx run --insecure

# Set a 10s timeout on every test
cx run --timeout 10s
```

---

## cx validate

Parse and schema-check test files without firing any HTTP requests or database queries.

```
cx validate [file or directory]
```

Exits with code `2` if any file fails validation.

### Examples

```bash
# Validate all files in current directory
cx validate

# Validate a specific file
cx validate tests/orders.cx.yaml
```

---

## cx init

Scaffold a commented `crosscheck.cx.yaml` template in the current directory.

```
cx init
```

---

## cx explain

Print a plain-English summary of a test file.

```
cx explain <file>
```

### Examples

```bash
cx explain tests/orders.cx.yaml
```

---

## Exit codes

| Code | Meaning                                        |
| ---- | ---------------------------------------------- |
| `0`  | All tests passed                               |
| `1`  | One or more tests failed                       |
| `2`  | YAML validation / config error                 |
| `3`  | Connection error (database or service adapter) |

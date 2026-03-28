# Global Config Reference

`.crosscheck.yaml` sets project-wide defaults for `cx run`. CLI flags always take precedence.

---

## Location

crosscheck looks for `.crosscheck.yaml` in the current working directory by default. Pass a custom path with `--config`:

```bash
cx run --config config/crosscheck.staging.yaml
```

A missing `.crosscheck.yaml` is silently ignored. If you pass `--config` explicitly and the file does not exist, an error is returned.

---

## All options

```yaml
# .crosscheck.yaml

# Reporter format used when --reporter is not passed.
# Accepted values: pretty, json, junit, html
reporter: pretty

# Default per-test timeout applied when a test has no timeout: field.
# Accepts any Go duration string: 10s, 500ms, 1m, etc.
timeout: 10s

# Skip TLS certificate verification (equivalent to --insecure).
insecure: false

# Path to the .env file (equivalent to --env-file).
env-file: .env
```

---

## Priority

CLI flags always override `.crosscheck.yaml`. Only flags that are **not explicitly set** on the command line are filled from the config file.

```
--reporter json       → uses json  (CLI wins)
reporter: junit       → uses junit (only if --reporter not passed)
```

---

## Example setups

**Monorepo with a shared `.env`:**

```yaml
env-file: config/.env.local
reporter: pretty
timeout: 15s
```

**CI environment:**

```yaml
reporter: junit
timeout: 30s
```

Run with:

```bash
cx run --config .crosscheck.ci.yaml
```

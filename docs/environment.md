# Environment Priority Reference

crosscheck merges variables from four sources. When the same key appears in multiple sources, the **highest-priority source wins**.

```
Priority (highest → lowest)
─────────────────────────────────────────────
1.  --env KEY=VALUE   CLI flag overrides
2.  Shell exports      process environment
3.  .env file          --env-file path
4.  YAML env: block    per-file defaults
─────────────────────────────────────────────
```

---

## 1. `--env` CLI flags (highest)

```bash
cx run --env BASE_URL=http://staging.example.com --env DEBUG=true
```

Repeatable. Always takes precedence over everything else.

---

## 2. Shell environment

```bash
export BASE_URL=http://staging.example.com
cx run
```

Standard shell exports. Overrides `.env` file and YAML defaults.

---

## 3. `.env` file

```bash
# .env
BASE_URL=http://localhost:3000
API_KEY=dev-secret
```

Loaded from the path given to `--env-file` (default: `.env` in the current directory). Missing file is silently ignored.

```bash
cx run --env-file .env.staging
```

---

## 4. YAML `env:` block (lowest)

```yaml
env:
  BASE_URL: http://localhost:3000   # used only if no other source sets BASE_URL
  TIMEOUT: 30s
```

Fallback defaults baked into the test file. Useful for documenting what variables the file expects and providing safe development defaults.

---

## Example — same key at multiple levels

Given:
- `.env`: `BASE_URL=http://localhost:3000`
- Shell: `export BASE_URL=http://staging.example.com`
- YAML: `env: { BASE_URL: http://dev.internal }`

The effective value is `http://staging.example.com` (shell wins over `.env` and YAML).

Add `--env BASE_URL=http://prod.example.com` and it becomes `http://prod.example.com`.

---

## Adapter connection URLs

The following environment variables activate optional adapters. They follow the same priority rules and can be set in any source:

| Variable | Adapter |
|---|---|
| `POSTGRES_URL` | Postgres |
| `MYSQL_URL` | MySQL / MariaDB |
| `MONGODB_URL` | MongoDB |
| `REDIS_URL` | Redis |
| `AWS_REGION` | SQS, SNS, S3, DynamoDB, Lambda |

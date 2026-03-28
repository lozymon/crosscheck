# CI/CD Integration Guide

## GitHub Actions

### Basic example

```yaml
# .github/workflows/api-tests.yml
name: API tests

on:
  push:
    branches: [main]
  pull_request:

jobs:
  test:
    runs-on: ubuntu-latest

    steps:
      - uses: actions/checkout@v4

      - name: Install crosscheck
        run: curl -sSL https://raw.githubusercontent.com/lozymon/crosscheck/main/install.sh | sh

      - name: Start services
        run: docker compose up -d --wait

      - name: Run tests
        env:
          BASE_URL: http://localhost:3000
          MYSQL_URL: root:root@tcp(localhost:3306)/cx
          REDIS_URL: redis://localhost:6379
        run: cx run --reporter junit --output-file results.xml

      - name: Publish test results
        uses: mikepenz/action-junit-report@v4
        if: always()
        with:
          report_paths: results.xml
```

---

### With secrets

Store sensitive values in GitHub Actions secrets and pass them as env vars:

```yaml
- name: Run tests
  env:
    BASE_URL: ${{ secrets.STAGING_URL }}
    API_KEY: ${{ secrets.API_KEY }}
    POSTGRES_URL: ${{ secrets.POSTGRES_URL }}
  run: cx run --reporter junit
```

---

### Full example — users-api

```yaml
name: users-api tests

on: [push, pull_request]

jobs:
  test:
    runs-on: ubuntu-latest

    steps:
      - uses: actions/checkout@v4

      - name: Install crosscheck
        run: curl -sSL https://raw.githubusercontent.com/lozymon/crosscheck/main/install.sh | sh

      - name: Start stack
        working-directory: examples/users-api
        run: docker compose up --build -d --wait

      - name: Run tests
        env:
          MYSQL_URL: root:root@tcp(localhost:3306)/cx
          REDIS_URL: redis://localhost:6379
          BASE_URL: http://localhost:3000
        run: cx run examples/users-api/tests/ --reporter junit --output-file results.xml

      - name: Publish results
        uses: mikepenz/action-junit-report@v4
        if: always()
        with:
          report_paths: results.xml

      - name: Stop stack
        if: always()
        working-directory: examples/users-api
        run: docker compose down -v
```

---

## Exit codes

crosscheck exits with a non-zero code on failure, which causes CI to mark the step as failed:

| Code | Meaning                                |
| ---- | -------------------------------------- |
| `0`  | All tests passed                       |
| `1`  | One or more tests failed               |
| `2`  | Config / YAML validation error         |
| `3`  | Connection error (database or service) |

---

## Tips

- Use `--reporter junit` in CI for rich test result visualisation.
- Use `--timeout` to prevent hung tests from blocking the pipeline: `cx run --timeout 30s`.
- Run `cx validate` as a pre-check step to catch YAML errors before firing any requests.
- Store `BASE_URL` and credentials in CI secrets — never commit them to the repo.

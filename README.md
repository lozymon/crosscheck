# crosscheck

End-to-end API testing from a single YAML file. Assert HTTP responses, database state, queues, caches, and outbound webhooks — all in one test run.

```yaml
version: 1
name: order flow

tests:
  - name: create order
    request:
      method: POST
      url: "{{ BASE_URL }}/orders"
      body: { productId: abc }
    response:
      status: 201
      body:
        id: "{{ capture: orderId }}"

    database:
      - adapter: mysql
        query: "SELECT status FROM orders WHERE id = :orderId"
        params: { orderId: "{{ orderId }}" }
        expect:
          - status: pending

    services:
      - adapter: redis
        key: "order:{{ orderId }}"
        expect: { status: pending }

      - adapter: mock
        path: /webhook
        method: POST
        wait_for: { timeout: 5s, interval: 200ms }
        expect: { event: order.created }
```

---

## Install

```bash
curl -sSL https://raw.githubusercontent.com/lozymon/crosscheck/main/install.sh | sh
```

Or with Go:

```bash
go install github.com/lozymon/crosscheck@latest
```

---

## Usage

```bash
cx run                              # find all *.cx.yaml recursively
cx run tests/orders.cx.yaml         # run a specific file
cx run tests/                       # run a directory
cx run --env BASE_URL=http://staging.example.com
cx run --env-file .env.staging
cx run --filter "order*"            # run matching tests only
cx run --reporter junit             # pretty | json | junit | html
cx run --watch                      # re-run on file changes
cx validate                         # schema check, no requests fired
cx explain tests/orders.cx.yaml     # plain-English summary
```

---

## Features

- **HTTP assertions** — status, headers, body (exact or partial)
- **Variable capture** — extract values from responses and chain into subsequent tests
- **Auth** — static header or login flow, runs once and injects into all tests
- **Database assertions** — Postgres, MySQL/MariaDB, MongoDB
- **Service assertions** — Redis, SQS, SNS, S3, DynamoDB, Lambda
- **Mock server** — capture outbound webhook calls and assert on them
- **Async polling** — `wait_for` retries assertions until they pass or timeout
- **Retry** — per-test `retry` + `retry_delay` for flaky flows
- **Setup / teardown** — shell hooks at file and test level
- **Reporters** — pretty, JSON, JUnit XML, HTML
- **Watch mode** — re-run on file save
- **Global config** — `.crosscheck.yaml` for project-wide defaults

---

## Adapters

Enable adapters by setting environment variables before running `cx`:

| Adapter | Environment variable |
|---|---|
| Postgres | `POSTGRES_URL` |
| MySQL / MariaDB | `MYSQL_URL=user:pass@tcp(host:port)/db` |
| MongoDB | `MONGODB_URL` |
| Redis | `REDIS_URL=redis://localhost:6379` |
| SQS / SNS / S3 / DynamoDB / Lambda | `AWS_REGION` |

---

## Example project

[`examples/users-api/`](examples/users-api/) is a complete working example: Node.js Express app backed by MariaDB and Redis, with a full crosscheck test suite.

```bash
cd examples/users-api
./run-tests.sh
```

---

## Documentation

- [Getting Started](docs/getting-started.md)
- [CLI Reference](docs/cli-reference.md)
- [YAML Schema](docs/yaml-schema.md)
- [Adapters](docs/adapters.md)
- [Auth](docs/auth.md)
- [Variables & Interpolation](docs/variables.md)
- [Environment Priority](docs/environment.md)
- [Global Config](docs/global-config.md)
- [Reporters](docs/reporters.md)
- [Watch Mode](docs/watch-mode.md)
- [CI/CD Integration](docs/ci-cd.md)
- [FAQ & Troubleshooting](docs/faq.md)

---

## License

MIT

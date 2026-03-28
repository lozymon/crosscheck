# Adapter Guides

Adapters connect crosscheck to databases and services so you can assert on state after an HTTP request.

- [Postgres](#postgres)
- [MySQL / MariaDB](#mysql--mariadb)
- [MongoDB](#mongodb)
- [Redis](#redis)
- [SQS](#sqs)
- [SNS](#sns)
- [S3](#s3)
- [DynamoDB](#dynamodb)
- [Lambda](#lambda)
- [Mock server](#mock-server)

---

## Postgres

**Activate:** set `POSTGRES_URL` environment variable.

```yaml
database:
  - adapter: postgres
    query: 'SELECT status, total FROM orders WHERE id = :orderId'
    params:
      orderId: '{{ orderId }}'
    wait_for:
      timeout: 10s
      interval: 500ms
    expect:
      - status: pending
        total: '99.99'
```

- Query params use `:varName` syntax — converted to `$1` placeholders automatically.
- `expect` is a list of rows. Each map is checked against the corresponding result row.
- `wait_for` polls until all rows match or the timeout elapses.
- All values are compared as strings.

---

## MySQL / MariaDB

**Activate:** set `MYSQL_URL` environment variable in DSN format: `user:pass@tcp(host:port)/dbname`

```yaml
database:
  - adapter: mysql
    query: 'SELECT name, email FROM users WHERE id = :userId'
    params:
      userId: '{{ userId }}'
    expect:
      - name: Alice
        email: alice@example.com
```

- Same `:varName` param syntax as Postgres — converted to `?` placeholders.
- Supports `wait_for` polling.

---

## MongoDB

**Activate:** set `MONGODB_URL` environment variable (standard MongoDB URI). The database name is taken from the URI path.

```yaml
database:
  - adapter: mongodb
    query: users # collection name
    params:
      email: alice@example.com # filter document
    expect:
      - name: Alice
        role: admin
```

- `query` is the collection name.
- `params` is the filter document (field equality).
- Returns all matching documents; each is checked against the corresponding `expect` row.

---

## Redis

**Activate:** set `REDIS_URL` environment variable, e.g. `redis://localhost:6379`.

```yaml
services:
  - adapter: redis
    key: 'user:{{ userId }}'
    expect:
      name: Alice
      email: alice@example.com
```

- If the key holds a plain string, it is returned as `{"value": "<string>"}`.
- If the key holds a hash (`HSET`), all fields are returned.
- If the key holds a JSON string, it is parsed and fields are compared directly.
- Supports `wait_for` polling.

---

## SQS

**Activate:** set `AWS_REGION` environment variable. Credentials come from the standard AWS credential chain (env, `~/.aws/credentials`, instance role).

```yaml
services:
  - adapter: sqs
    queue: https://sqs.us-east-1.amazonaws.com/123456789/my-queue
    wait_for:
      timeout: 15s
      interval: 1s
    expect:
      eventType: order.created
      orderId: '{{ orderId }}'
```

- Uses a non-destructive peek (`VisibilityTimeout=0`) — messages are not consumed.
- **Any-match semantics:** at least one message in the queue must satisfy all `expect` fields.
- JSON message bodies are parsed; fields are compared as strings.

---

## SNS

**Activate:** set `AWS_REGION` environment variable.

SNS assertions work by reading from an SQS queue subscribed to the SNS topic. crosscheck unwraps the SNS notification envelope automatically.

```yaml
services:
  - adapter: sns
    queue: https://sqs.us-east-1.amazonaws.com/123456789/my-sns-subscriber-queue
    wait_for:
      timeout: 15s
      interval: 1s
    expect:
      event: order.shipped
```

- `queue` is the URL of an SQS queue subscribed to the SNS topic — not the topic ARN.
- The SNS envelope `Message` field is unwrapped; if it is JSON it is parsed for field assertions.
- If `Message` is a plain string it is returned as `{"message": "<string>"}`.

---

## S3

**Activate:** set `AWS_REGION` environment variable.

```yaml
services:
  - adapter: s3
    bucket: my-exports-bucket
    key: 'reports/{{ reportId }}.json'
    wait_for:
      timeout: 30s
      interval: 2s
    expect:
      status: complete
      recordCount: '42'
```

- Downloads the object and JSON-parses the body.
- `expect` fields are compared against the parsed JSON.
- Supports `wait_for` polling.

---

## DynamoDB

**Activate:** set `AWS_REGION` environment variable.

```yaml
services:
  - adapter: dynamodb
    table: Orders
    key: '{{ orderId }}' # partition key value
    key_name: orderId # partition key attribute name (default: "id")
    sort_key: '2024-01-01' # optional sort key value
    sort_key_name: createdAt # sort key attribute name
    wait_for:
      timeout: 10s
      interval: 500ms
    expect:
      status: shipped
      total: '99.99'
```

- Uses `GetItem` — asserts a single item by primary key.
- All attribute values are stringified for comparison.

---

## Lambda

**Activate:** set `AWS_REGION` environment variable.

```yaml
services:
  - adapter: lambda
    key: my-function-name # function name or ARN
    payload:
      userId: '{{ userId }}'
      action: notify
    expect:
      statusCode: '200'
      message: ok
```

- Invokes the function synchronously (`RequestResponse`).
- `payload` is the invocation input (serialised as JSON).
- The function response body is JSON-parsed and compared against `expect`.
- Does not support `wait_for` (invocation is synchronous).

---

## Mock server

The mock server captures outbound HTTP calls made by your application so you can assert they happened.

**Activate:** add a `mock:` block to your test file.

```yaml
mock:
  port: 9099 # 0 = auto-assign
```

`MOCK_URL` is injected automatically — pass it to your app as `WEBHOOK_URL` or similar.

```yaml
services:
  - adapter: mock
    path: /webhook # URL path filter (empty = any path)
    method: POST # method filter (empty = any method)
    wait_for:
      timeout: 5s
      interval: 200ms
    expect:
      event: order.created
      orderId: '{{ orderId }}'
```

- **Any-match semantics:** at least one captured request matching `path`/`method` must satisfy all `expect` fields.
- JSON request bodies are parsed; fields are compared as strings.
- Use `wait_for` for async webhooks that may arrive slightly after the HTTP response.
- The mock server responds `200 OK` with an empty body to all requests.

# order-pipeline

A multi-service e-commerce order pipeline used as a realistic crosscheck example.
It exercises HTTP assertions, MariaDB polling, Redis cache checks, RabbitMQ queue
inspection, and webhook capture — all from a single `cx run` command.

## Architecture

```
 ┌──────────────┐  POST /orders   ┌─────────────────────────────────┐
 │   cx test    │ ──────────────► │          order-api :3000         │
 └──────────────┘                 │  MariaDB • RabbitMQ publisher    │
        │                         └──────────────┬──────────────────┘
        │  assert HTTP / DB / Redis / mock        │ order.placed event
        │                              ┌──────────┴──────────┐
        │                              ▼                      ▼
        │                  ┌───────────────────┐  ┌───────────────────────┐
        │                  │ inventory-service │  │ notification-service  │
        │                  │ MariaDB + Redis   │  │ webhook → mock server │
        │                  │ + low-stock hook  │  └───────────────────────┘
        │                  └───────────────────┘
        │
        └─► mock server :9099  (captures /low-stock and /notify webhooks)
```

**Services**

| Service                | Role                                                            |
| ---------------------- | --------------------------------------------------------------- |
| `order-api`            | REST API — validates stock, persists order, publishes event     |
| `inventory-service`    | Consumes queue, updates Redis stock cache, fires low-stock hook |
| `notification-service` | Consumes queue, fires `/notify` webhook per order               |

**Infrastructure**

| Service   | Internal port | Host port    |
| --------- | ------------- | ------------ |
| MariaDB   | 3306          | 3307         |
| Redis     | 6379          | 6380         |
| RabbitMQ  | 5672 / 15672  | 5672 / 15672 |
| order-api | 3000          | 4000         |

## Prerequisites

- Docker + Docker Compose v2
- [crosscheck](https://github.com/lozymon/crosscheck) installed (`cx` on `$PATH`)

## Running the tests

```bash
# 1. Start all services (builds images on first run)
docker compose up -d --build --wait

# 2. Run both test suites from this directory
cx run ./tests/

# 3. Tear down
docker compose down
```

Or use the convenience script:

```bash
./run-tests.sh
```

## Test suites

### `tests/place-order.cx.yaml` — Happy path

1. `POST /orders` → asserts HTTP 201, captures `orderId` + `productId`
2. MariaDB — `orders` row exists with `status: placed`
3. RabbitMQ — `order.placed` event present in `assert-orders` queue
4. MariaDB — `products.stock` decremented (polled with `wait_for`)
5. Redis — `stock:<productId>` key reflects new stock (polled)
6. Mock server — `POST /low-stock` webhook received (stock dropped to 1 ≤ threshold 2)
7. Mock server — `POST /notify` webhook received with correct `orderId`

> Setup resets product stock to **2** so the single placed order tips it below
> `LOW_STOCK_THRESHOLD` (also 2), triggering the low-stock alert path.

### `tests/out-of-stock.cx.yaml` — Rejection path

1. Setup sets stock to 0
2. `POST /orders` → asserts HTTP 409 `{ error: "out of stock" }`
3. MariaDB — confirms no `orders` row was inserted

## RabbitMQ topology

The `rabbitmq/definitions.json` file pre-declares the following topology:

```
Exchange: order-events (topic)
   │
   ├─► inventory-orders    — consumed by inventory-service
   ├─► notification-orders — consumed by notification-service
   └─► assert-orders       — NOT consumed; exists only for cx assertions
```

`assert-orders` is a mirror queue that all `order.#` messages copy into.
Because nothing consumes it, messages stay in the queue until the test
setup purges it, giving crosscheck a stable snapshot to assert against.

## RabbitMQ management UI

Open <http://localhost:15672> (guest / guest) to inspect queues and messages
while the services are running.

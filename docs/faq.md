# FAQ & Troubleshooting

## Tests pass locally but fail in CI

**Check environment variables.** CI doesn't have your local `.env` file. Pass secrets via CI environment variables or secrets manager.

**Check service URLs.** `localhost` in CI often refers to the CI runner itself, not a Docker container. Use the service name (e.g. `mariadb`) or `127.0.0.1` depending on how your CI network is configured.

**Check timeouts.** CI machines can be slower than local. Increase `wait_for` timeouts or set a global `timeout` in `.crosscheck.yaml`.

---

## `adapter not configured` error

The adapter URL environment variable is not set. Check:

| Error | Variable to set |
|---|---|
| `postgres adapter not configured` | `POSTGRES_URL` |
| `mysql adapter not configured` | `MYSQL_URL` |
| `mongodb adapter not configured` | `MONGODB_URL` |
| `redis adapter not configured` | `REDIS_URL` |
| `sqs/sns/s3/dynamodb/lambda adapter not configured` | `AWS_REGION` |

These can be set in your shell, `.env` file, or with `--env`.

---

## `no *.cx.yaml test files found`

crosscheck looks for files matching `*.cx.yaml` recursively. Make sure your files:
- Have the `.cx.yaml` extension (not just `.yaml`)
- Are in the path you passed to `cx run`

---

## `status: expected "201", got "500"`

Your application returned an error. Check:
1. Application logs for the actual error.
2. That the service is running and the URL is correct.
3. That the request body is valid (missing required fields, wrong Content-Type header).

---

## Captured variable is empty

If `{{ capture: myVar }}` results in an empty string:
1. Check the response body matches the expected JSON structure.
2. The capture extracts `$.key` from the body — make sure the key exists at the top level.
3. Print the raw response by temporarily adding `--reporter json` to see what the API actually returns.

---

## Mock server receives no requests

1. **Binding issue** — the mock server listens on `0.0.0.0` so it should be reachable from Docker. Verify `host.docker.internal` resolves inside your container: `docker compose exec app getent hosts host.docker.internal`.
2. **Timing** — add or increase `wait_for` on the mock assertion to give the app time to fire the webhook.
3. **WEBHOOK_URL not set** — make sure your app is configured with the correct mock server URL.
4. **Webhook errors** — check your application logs for outbound HTTP errors.

---

## `wait_for` never passes

1. The assertion condition is wrong — check the `expect` fields match what's actually in the database/queue.
2. The service is not processing the request — check application logs.
3. Increase `timeout` — default is whatever you set; there is no built-in fallback.

---

## `cx` command not found after install

Make sure `$(go env GOPATH)/bin` is in your `PATH`:

```bash
export PATH="$PATH:$(go env GOPATH)/bin"
```

Add this line to your `~/.zshrc` or `~/.bashrc` to make it permanent.

---

## TLS certificate errors

Use `--insecure` to skip TLS verification for self-signed certificates:

```bash
cx run --insecure
```

Or set it permanently in `.crosscheck.yaml`:

```yaml
insecure: true
```

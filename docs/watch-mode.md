# Watch Mode

Watch mode re-runs tests automatically whenever a `*.cx.yaml` file changes.

```bash
cx run --watch
cx run tests/ --watch
```

---

## How it works

1. crosscheck runs all discovered test files immediately on startup.
2. It then watches the test files and their parent directories for changes.
3. When a change is detected, crosscheck waits 300ms (to coalesce rapid saves) and re-runs all tests.
4. Output is printed fresh on each run.

---

## Use cases

- **TDD loop** — edit your test file and see results instantly.
- **Live feedback** — keep a terminal open while developing an API.

---

## Notes

- Watch mode monitors `*.cx.yaml` files in the given path, not your application source.
- To also restart your application on changes, run your app with its own watch tool (e.g. `nodemon`) in a separate terminal.
- Press `Ctrl+C` to stop.

# Reporter Guide

Select a reporter with `--reporter <format>` or set a default in `.crosscheck.yaml`.

---

## pretty (default)

Coloured terminal output. Pass/fail per test with failure details.

```bash
cx run
cx run --reporter pretty
```

```
order suite

  ✓  create order
  ✗  fetch order
     response: status: expected "200", got "404"

  2 tests  1 passed  1 failed
```

---

## json

Single JSON object written to stdout. Useful for piping into other tools.

```bash
cx run --reporter json
```

```json
{
  "suite": "order suite",
  "passed": 1,
  "failed": 1,
  "tests": [
    { "name": "create order", "passed": true },
    {
      "name": "fetch order",
      "passed": false,
      "failures": [
        {
          "step": "response",
          "message": "status: expected \"200\", got \"404\""
        }
      ]
    }
  ]
}
```

---

## junit

JUnit XML — compatible with GitHub Actions, Jenkins, GitLab CI, and most CI systems.

```bash
cx run --reporter junit
```

When running multiple test files, all suites are combined into a single `<testsuites>` document.

Setup errors appear as `<error>` elements; test failures as `<failure>` elements.

---

## html

Self-contained single-file HTML report with no external dependencies.

```bash
cx run --reporter html > report.html
```

- Collapsible test rows — failing tests are expanded automatically.
- Summary totals at the top.
- Works offline.

---

## `--output-file`

Write JSON results to a file in addition to the selected reporter on stdout:

```bash
cx run --reporter pretty --output-file results.json
```

The file receives the same JSON structure as `--reporter json`.

---

## Setting a default reporter

In `.crosscheck.yaml`:

```yaml
reporter: junit
```

CLI `--reporter` always overrides this.

#!/usr/bin/env sh
set -e

echo "==> Starting services..."
docker compose up -d --build --wait

echo "==> Running test suites..."
cx run ./tests/
EXIT_CODE=$?

echo "==> Tearing down..."
docker compose down

exit $EXIT_CODE

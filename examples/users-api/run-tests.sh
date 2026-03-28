#!/bin/sh
set -e

# Start the stack and wait for all health checks to pass.
docker compose -f docker-compose.yml up --build -d --wait

# Run the crosscheck test suite.
../../cx run tests/ --env-file .env

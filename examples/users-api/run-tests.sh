#!/bin/sh
set -e

COMPOSE="docker compose -f docker-compose.yml"

# Start the stack and wait for all health checks to pass.
$COMPOSE up --build -d --wait

# Run the crosscheck test suite.
../../cx run tests/ --env-file .env

#!/bin/sh
set -eu

docker compose -f docker-compose.yml logs -f

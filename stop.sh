#!/bin/bash
set -e
cd "$(dirname "$0")"

podman-compose -f docker-compose.prod.yml down
echo "✓ Service stopped!"

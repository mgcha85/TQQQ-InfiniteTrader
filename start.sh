#!/bin/bash
set -e
cd "$(dirname "$0")"

# Ensure data directory exists for volume mount
mkdir -p data

# Always replace existing containers so new image tags are applied.
podman-compose -f docker-compose.prod.yml down || true
podman-compose -f docker-compose.prod.yml up -d --force-recreate --remove-orphans
echo "✓ Service started!"

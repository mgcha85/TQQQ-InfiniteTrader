#!/bin/bash
set -e
cd "$(dirname "$0")"

# Ensure data directory exists for volume mount
mkdir -p data

podman-compose -f docker-compose.prod.yml up -d --remove-orphans
echo "✓ Service started!"

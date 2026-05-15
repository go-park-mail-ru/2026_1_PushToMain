#!/bin/bash
set -e
cd ~/backend

echo "$2" | docker login ghcr.io -u "$1" --password-stdin

docker compose -f docker-compose.prod.yml pull
docker compose -f docker-compose.prod.yml down
docker compose -f docker-compose.prod.yml up -d --remove-orphans
docker image prune -f

#!/bin/bash
set -e

cd ~/backend

rm -f .env
echo "POSTGRES_USER=$1" >> .env
echo "POSTGRES_PASSWORD=$2" >> .env
echo "POSTGRES_DBNAME=$3" >> .env
echo "MINIO_ACCESS_KEY=$4" >> .env
echo "MINIO_SECRET_KEY=$5" >> .env
echo "JWT_SECRET=$6" >> .env

sudo docker compose -f docker-compose.prod.yml pull
sudo docker compose -f docker-compose.prod.yml up -d

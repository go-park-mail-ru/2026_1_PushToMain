#!/bin/bash
set -e

mkdir -p certs/postfix
openssl req -x509 -newkey rsa:4096 \
  -keyout certs/postfix/privkey.pem \
  -out certs/postfix/fullchain.pem \
  -days 365 -nodes \
  -subj "/CN=mail.e-smail.ru"

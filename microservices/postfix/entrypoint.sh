#!/bin/bash
set -e

postmap /etc/postfix/transport

exec postfix start-fg

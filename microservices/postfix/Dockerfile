FROM ubuntu:24.04

RUN apt-get update && \
    apt-get install -y postfix && \
    rm -rf /var/lib/apt/lists/*

COPY microservices/postfix/entrypoint.sh /entrypoint.sh
RUN chmod +x /entrypoint.sh

COPY configs/postfix/transport /etc/postfix/transport
RUN postmap /etc/postfix/transport

ENTRYPOINT ["/entrypoint.sh"]

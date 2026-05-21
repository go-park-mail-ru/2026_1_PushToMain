FROM ubuntu:24.04

RUN apt-get update && \
    apt-get install -y postfix && \
    rm -rf /var/lib/apt/lists/*

COPY docker/postfix/entrypoint.sh /entrypoint.sh
RUN chmod +x /entrypoint.sh

ENTRYPOINT ["/entrypoint.sh"]

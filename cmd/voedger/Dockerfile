# Copyright (c) 2024 Sigma-Soft, Ltd.
# @author Aleksei Ponomarev

FROM ubuntu:20.04

RUN apt update && \
    apt install -y ca-certificates

RUN mkdir /app
COPY voedger /app

COPY entrypoint.sh /app
RUN chmod +x /app/entrypoint.sh

ENV VOEDGER_HTTP_PORT 443
ENV VOEDGER_ACME_DOMAINS ""
ENV VOEDGER_STORAGE_TYPE cas3

# This container exposes port to the outside world
EXPOSE $VOEDGER_HTTP_PORT

# Set the Current Working Directory inside the container
WORKDIR /app

# Use the script as entrypoint
ENTRYPOINT ["/app/entrypoint.sh"]

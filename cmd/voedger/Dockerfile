FROM ubuntu:20.04

RUN apt update && \
    apt install -y ca-certificates

RUN mkdir /app
COPY voedger /app

ENV VOEDGER_HTTP_PORT 443
ENV VOEDGER_ACME_DOMAINS ""

# This container exposes port to the outside world
EXPOSE $VOEDGER_HTTP_PORT

# Set the Current Working Directory inside the container
WORKDIR /app

# Use ENTRYPOINT to conditionally generate command based on env var
ENTRYPOINT ["/bin/bash", "-c", \
  "if [ -n \"$VOEDGER_ACME_DOMAINS\" ]; then \
    IFS=',' read -ra domains <<< \"$VOEDGER_ACME_DOMAINS\" && \
    cmd_args=(); \
    for domain in \"${domains[@]}\"; do \
      cmd_args+=(\"--acme-domain\" \"$domain\"); \
    done; \
    exec /app/voedger server --ihttp.Port=$VOEDGER_HTTP_PORT --storage cas3 \"${cmd_args[@]}\" \"$@\"; \
  else \
    exec /app/voedger server --ihttp.Port=$VOEDGER_HTTP_PORT --storage cas3 \"$@\"; \
  fi"]


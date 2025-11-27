#!/usr/bin/env bash
#
# Copyright (c) 2024 Sigma-Soft, Ltd.
# @author Aleksei Ponomarev
#

set -Eeuo pipefail

# Default storage type
storage_type="${VOEDGER_STORAGE_TYPE:-cas3}"

# Initial command arguments
cmd_args=("--ihttp.Port=$VOEDGER_HTTP_PORT" "--storage" "$storage_type")

# Check if VOEDGER_ACME_DOMAINS is set and not empty
if [ -n "$VOEDGER_ACME_DOMAINS" ]; then
    IFS=',' read -ra domains <<< "$VOEDGER_ACME_DOMAINS"
    for domain in "${domains[@]}"; do
        cmd_args+=("--acme-domain" "$domain")
    done
fi

# Add logic for additional environment variables here
# Example:
# if [ -n "$NEW_ENV_VAR" ]; then
#     cmd_args+=("--new-option" "$NEW_ENV_VAR")
# fi

# Execute the command
exec /app/voedger server "${cmd_args[@]}" "$@"

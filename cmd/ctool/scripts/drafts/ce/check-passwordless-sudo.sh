#!/usr/bin/env bash
#
# Check if passwordless sudo is configured on remote Ubuntu server
#
set -euo pipefail

if [ $# -ne 1 ]; then
    echo "Usage: $0 <node>"
    exit 1
fi

source ../utils.sh

NODE=$1
SSH_USER=${SSH_USER:-$LOGNAME}

# Set default SSH key if not provided
if [ -z "${VOEDGER_SSH_KEY:-}" ]; then
    export VOEDGER_SSH_KEY="${HOME}/.ssh/id_rsa"
fi

# Check if passwordless sudo is configured by trying to run a simple sudo command
utils_ssh "$SSH_USER@$NODE" "sudo -n true" 2>/dev/null

if [ $? -eq 0 ]; then
    echo "Passwordless sudo is configured on CE node $NODE"
    exit 0
else
    echo "Passwordless sudo is not configured on CE node $NODE"
    exit 1
fi

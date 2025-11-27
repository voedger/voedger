#!/usr/bin/env bash
#
# Check if passwordless sudo is configured on remote Ubuntu server
#
set -Eeuo pipefail

if [ $# -ne 1 ]; then
    echo "Usage: $0 <node>"
    exit 1
fi

source ../utils.sh

NODE=$1

# Check if passwordless sudo is configured by trying to run a simple sudo command
sudo -n true 2>/dev/null

if [ $? -eq 0 ]; then
    echo "Passwordless sudo is configured on CE node $NODE"
    exit 0
else
    echo "Passwordless sudo is not configured on CE node $NODE"
    exit 1
fi

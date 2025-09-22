#!/usr/bin/env bash
#
# Copyright (c) 2024 Sigma-Soft, Ltd.
# @author Dmitry Molchanovsky
#
#
# Sets a new password for the admin user in Prometheus
# on specified CE node via SSH
set -euo pipefail
set -x
if [ $# -ne 3 ]; then
    echo "Usage: $0 <password> <hashed password> <node>"
    exit 1
fi

source ../utils.sh

NEW_PASSWORD=$1
HASHED_PASSWORD=$2
NODE=$3
SSH_USER=${SSH_USER:-$LOGNAME}
USER_NAME="voedger"

echo "Setting Prometheus password for user '$USER_NAME' on CE node $NODE..."

# Check if Prometheus container exists on remote node
APP_NODE_CONTAINER=$(utils_ssh "$SSH_USER@$NODE" "sudo docker ps --format '{{.Names}}' | grep prometheus" || echo "")
if [ -z "$APP_NODE_CONTAINER" ]; then
    echo "Prometheus container was not found on CE node $NODE"
    exit 1
else
    echo "Found Prometheus container: $APP_NODE_CONTAINER"

    # Update the password in web.yml on remote node
    ESCAPED_HASHED_PASSWORD=$(echo "$HASHED_PASSWORD" | sed 's/[\/&]/\\&/g')
    utils_ssh "$SSH_USER@$NODE" "sed -i 's/${USER_NAME}:.*/${USER_NAME}: $ESCAPED_HASHED_PASSWORD/' ~/prometheus/web.yml"
    echo "Password for voedger user in Prometheus on CE node $NODE was successfully changed"

    # Restart Prometheus container on remote node
    utils_ssh "$SSH_USER@$NODE" "sudo docker restart $APP_NODE_CONTAINER"
    echo "Prometheus container was restarted on CE node $NODE"
fi

set +x

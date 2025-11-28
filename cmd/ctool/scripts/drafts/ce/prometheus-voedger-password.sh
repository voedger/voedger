#!/usr/bin/env bash
#
# Copyright (c) 2024 Sigma-Soft, Ltd.
# @author Dmitry Molchanovsky
#
#
# Sets a new password for the admin user in Prometheus
# on specified local hosts
set -Eeuo pipefail
set -x
if [ $# -ne 2 ]; then
    echo "Usage: $0 <password> <hashed password>"
    exit 1
fi

NEW_PASSWORD=$1
HASHED_PASSWORD=$2
USER_NAME="voedger"

APP_NODE_CONTAINER=$(sudo docker ps --format '{{.Names}}' | grep prometheus)
if [ -z "$APP_NODE_CONTAINER" ]; then
    echo "Prometheus container was not found"
else
    ESCAPED_HASHED_PASSWORD=$(echo "$HASHED_PASSWORD" | sed 's/[\/&]/\\&/g')
    sed -i "s/${USER_NAME}:.*/${USER_NAME}: $ESCAPED_HASHED_PASSWORD/" ~/prometheus/web.yml
    echo "Password for voedger user in Prometheus on ce-node was successfully changed"

    ./docker-container-restart.sh prometheus
    echo "Prometheus service was updated"
fi

set +x

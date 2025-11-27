#!/usr/bin/env bash
#
# Copyright (c) 2024 Sigma-Soft, Ltd.
# @author Dmitry Molchanovsky
#
#
# Sets a new password for the admin user in Grafana
# on the local machine
set -Eeuo pipefail
set -x

if [ $# -ne 1 ]; then
    echo "Usage: $0 <password>"
    exit 1
fi

NEW_PASSWORD=$1

# Find the Grafana container running locally
APP_NODE_CONTAINER=$(sudo docker ps --format '{{.Names}}' | grep grafana)

if [ -z "$APP_NODE_CONTAINER" ]; then
    echo "Grafana container was not found on the local machine"
    exit 1
fi

# Check if Grafana is running
while ! curl -s http://localhost:3000/login | grep -q 'Grafana'; do
    echo "Waiting for Grafana to start..."
    sleep 5
done

# Execute the command to reset the admin password
sudo docker exec ${APP_NODE_CONTAINER} grafana-cli admin reset-admin-password ${NEW_PASSWORD}
echo "Password for admin user in Grafana was successfully changed"

set +x
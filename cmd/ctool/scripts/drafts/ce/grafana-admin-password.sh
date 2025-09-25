#!/usr/bin/env bash
#
# Copyright (c) 2024 Sigma-Soft, Ltd.
# @author Dmitry Molchanovsky
# 
# 
# Sets a new password for the admin user in Grafana
# on the local machine
set -euo pipefail
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

# Check if Grafana is running with timeout
echo "Checking if Grafana is ready..."
TIMEOUT=300  # 5 minutes timeout
ELAPSED=0
INTERVAL=5

while [ $ELAPSED -lt $TIMEOUT ]; do
    # Check if Grafana responds to HTTP requests
    if curl -s --connect-timeout 5 --max-time 10 http://localhost:3000/api/health > /dev/null 2>&1; then
        echo "Grafana is ready!"
        break
    fi

    echo "Waiting for Grafana to start... (${ELAPSED}s/${TIMEOUT}s)"
    sleep $INTERVAL
    ELAPSED=$((ELAPSED + INTERVAL))
done

if [ $ELAPSED -ge $TIMEOUT ]; then
    echo "Timeout: Grafana did not start within ${TIMEOUT} seconds"
    echo "Checking Grafana container status:"
    sudo docker ps | grep grafana || echo "No Grafana container found"
    echo "Checking Grafana logs:"
    sudo docker logs ${APP_NODE_CONTAINER} --tail 20 || echo "Could not get container logs"
    exit 1
fi

# Execute the command to reset the admin password
sudo docker exec ${APP_NODE_CONTAINER} grafana-cli admin reset-admin-password ${NEW_PASSWORD}
echo "Password for admin user in Grafana was successfully changed"

set +x
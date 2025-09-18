#!/usr/bin/env bash
#
# Copyright (c) 2024 Sigma-Soft, Ltd.
# @author Dmitry Molchanovsky
#
#
# Sets a new password for the admin user in Grafana
# on the CE node via SSH
set -euo pipefail

if [ $# -ne 2 ]; then
    echo "Usage: $0 <password> <ce-node host>"
    exit 1
fi

source ../utils.sh

NEW_PASSWORD=$1
CE_NODE_HOST=$2
SSH_USER=$LOGNAME

echo "Setting Grafana admin password on CE node $CE_NODE_HOST..."

# Create the script to run on the remote CE node
script="\
# Find the Grafana container running on the CE node
APP_NODE_CONTAINER=\$(sudo docker ps --format '{{.Names}}' | grep grafana);
if [ -z \"\$APP_NODE_CONTAINER\" ]; then
    echo \"Grafana container was not found on the CE node\";
    exit 1;
fi;

# Check if Grafana is running
while ! curl -s http://localhost:3000/login | grep -q 'Grafana'; do
    echo \"Waiting for Grafana to start...\";
    sleep 5;
done;

# Execute the command to reset the admin password
sudo docker exec \${APP_NODE_CONTAINER} grafana-cli admin reset-admin-password $NEW_PASSWORD;
echo \"Password for admin user in Grafana was successfully changed\";
"

echo "Executing Grafana admin password reset on remote CE node..."
utils_ssh_interactive "$SSH_USER@$CE_NODE_HOST" "bash -s" << EOF
$script
EOF

echo "Grafana admin password successfully set on CE node."
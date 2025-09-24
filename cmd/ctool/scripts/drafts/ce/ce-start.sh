#!/usr/bin/env bash
#
# Copyright (c) 2024 Sigma-Soft, Ltd.
# @author Aleksei Ponomarev
#
# Deploy Voedger CE cluster

set -euo pipefail

if [ "$#" -lt 1 ]; then
  echo "Usage: $0 <node ip address for CE deployment>" >&2
  exit 1
fi

source ../utils.sh

NODE=$1
SSH_USER=$LOGNAME

echo "Deploying Voedger CE on remote host $NODE..."

# Get the CE node IP from environment variable
CE_NODE_IP="${VOEDGER_CE_NODE:-$NODE}"

# Create the CE deployment script to run on the remote machine
script="\
HOSTNAME=\"db-node-1\";
CE_NODE=\"$CE_NODE_IP\";
if [ -n \"\$CE_NODE\" ]; then
    # Update /etc/hosts
    if grep -q \"\$HOSTNAME\" /etc/hosts; then
        sudo sed -i\".bak\" \"/\$HOSTNAME/c\\\\\$CE_NODE \$HOSTNAME\" /etc/hosts;
    else
        sudo bash -c \"echo \\\"\$CE_NODE \$HOSTNAME\\\" >> /etc/hosts\";
    fi;

    # Set environment variables for Docker Compose
    export VOEDGER_CE_NODE=\"\$CE_NODE\";
    export VOEDGER_HTTP_PORT=\"${VOEDGER_HTTP_PORT:-80}\";
    export VOEDGER_ACME_DOMAINS=\"${VOEDGER_ACME_DOMAINS:-}\";

    # Start the Voedger CE Docker stack
    echo \"Starting Voedger CE Docker containers...\";
    envsubst < /tmp/docker-compose-ce.yml > /tmp/docker-compose-final.yml;
    sudo docker-compose -p CEDockerStack -f /tmp/docker-compose-final.yml up -d;

    echo \"Voedger CE deployment completed.\";
else
   echo \"Error deploy Voedger CE. CE_NODE not set.\";
   exit 1;
fi;
"

echo "Copying Docker Compose file to remote host..."
utils_scp "docker-compose-tmp.yml" "$SSH_USER@$NODE:/tmp/docker-compose-ce.yml"

echo "Starting Voedger CE deployment on remote host..."
utils_ssh_interactive "$SSH_USER@$NODE" "bash -s" << EOF
$script
EOF

echo "Voedger CE deployment completed on remote host."
                                                                                                             
                                                                                                             

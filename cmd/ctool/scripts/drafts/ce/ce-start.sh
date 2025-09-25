#!/usr/bin/env bash
#
# Copyright (c) 2024 Sigma-Soft, Ltd.
# @author Aleksei Ponomarev
#
# envsubst < ./docker-compose.yml > docker-compose-tmp.yml
# docker-compose -p voedger -f ./docker-compose-tmp.yml ps

set -euo pipefail

if [ "$#" -lt 1 ]; then
  echo "Usage: $0 <node ip address for CE deployment>" >&2
  exit 1
fi

source ../utils.sh

NODE=$1
HOSTNAME="db-node-1"
echo "Deploying Voedger CE on host $NODE..."
echo "Starting Voedger CE deployment on host..."
CE_NODE="$CE_NODE_IP";
if [ -n "$CE_NODE" ]; then
    # Update /etc/hosts
    if grep -q "$HOSTNAME" /etc/hosts; then
        sudo sed -i".bak" "/$HOSTNAME/c\\$CE_NODE $HOSTNAME" /etc/hosts;
    else
        sudo bash -c "echo \\"$CE_NODE $HOSTNAME\\" >> /etc/hosts";
    fi;

    # Set environment variables for Docker Compose
    export VOEDGER_CE_NODE="$CE_NODE";
    export VOEDGER_HTTP_PORT="${VOEDGER_HTTP_PORT:-80}";
    export VOEDGER_ACME_DOMAINS="${VOEDGER_ACME_DOMAINS:-}";

    # Start the Voedger CE Docker stack
    echo "Starting Voedger CE Docker containers...";
    echo "Copying Docker Compose file to host..."
    cp docker-compose-tmp.yml /tmp/docker-compose-ce.yml
    envsubst < /tmp/docker-compose-ce.yml > /tmp/docker-compose-final.yml;
    sudo docker-compose -p CEDockerStack -f /tmp/docker-compose-final.yml up -d;

    echo "Voedger CE deployment completed.";
else
   echo "Error deploy Voedger CE. Use export VOEDGER_CE_NODE= <hostname | ipaddress>."
   exit 1
fi


if [ -n "${VOEDGER_CE_NODE:-}" ]; then
    # Check if the record with db-node-1 is present
    if grep -q "$HOSTNAME" /etc/hosts; then
        # If present, update the record with the new IP
        sudo sed -i".bak" "/$HOSTNAME/c\\$VOEDGER_CE_NODE $HOSTNAME" /etc/hosts
    else
        # If not present, add the new record
        sudo bash -c "echo \"$VOEDGER_CE_NODE $HOSTNAME\" >> /etc/hosts"
    fi
    envsubst < ./docker-compose-tmp.yml > ./docker-compose.yml
    sudo docker-compose -p voedger  up -d
else
   echo "Error deploy Voedger CE. Use export VOEDGER_CE_NODE= <hostname | ipaddress>."
   exit 1
fi

echo "Voedger CE deployment completed on host."

                                                                                                             

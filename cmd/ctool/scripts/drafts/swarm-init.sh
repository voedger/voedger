#!/usr/bin/env bash
#
# Copyright (c) 2023 Sigma-Soft, Ltd.
# @author Aleksei Ponomarev
#
# Init Swarm if not already, check init by .SwarmLocalNodeState
#    - if inactive - init
#    - store token for workers to 'worker.token' file
#    - get token for add managers
#    - store token for manangers to 'manager.token' file

set -euo pipefail

set +x

if [ "$#" -lt 1 ]; then
  echo "Usage: $0 <ip address first swarm node>" >&2
  exit 1
fi

# Define the desired subnet and gateway
NEW_SUBNET="192.168.44.0/24"
NEW_GATEWAY="192.168.44.1"
swarm_manager_ip=$(getent hosts "$1" | awk '{print $1}')

SSH_USER=$LOGNAME
SSH_OPTIONS='-o UserKnownHostsFile=/dev/null -o StrictHostKeyChecking=no -o LogLevel=ERROR'

if [[ $(ssh $SSH_OPTIONS $SSH_USER@"$1" docker info --format '{{.Swarm.LocalNodeState}}') == "inactive" ]]; then
  # Initialize Swarm with all nodes as managers and workers
  WORKER_TOKEN=$(ssh $SSH_OPTIONS $SSH_USER@"$1" docker swarm init --advertise-addr "$swarm_manager_ip" --listen-addr "$swarm_manager_ip":2377 | grep -oP "SWMTKN-\S+")
  echo "$WORKER_TOKEN" > worker.token
fi

MANAGER_TOKEN=$(ssh $SSH_OPTIONS $SSH_USER@$1 docker swarm join-token manager | grep -oP "SWMTKN-\S+")
echo "$MANAGER_TOKEN" > manager.token

# Get the current subnet of the ingress network
CURRENT_SUBNET=$(ssh $SSH_OPTIONS $SSH_USER@$1 "docker network inspect -f '{{range .IPAM.Config}}{{.Subnet}}{{end}}' ingress")
echo "$CURRENT_SUBNET"

# Check if the current subnet matches the desired subnet
if [ "$CURRENT_SUBNET" == "10.0.0.0/24" ]; then
    # Remove the existing ingress network
  echo "Remove ingress network"
  ssh "$SSH_OPTIONS" "$SSH_USER"@"$1" "echo y | docker network rm ingress "

  echo "Create ingress network"
  # Create a new ingress network with the desired subnet
  ssh "$SSH_OPTIONS" "$SSH_USER"@"$1" "docker network create \
  --driver overlay \
  --ingress \
  --subnet=$NEW_SUBNET \
  --gateway=$NEW_GATEWAY \
  --opt com.docker.network.driver.overlay.vxlanid_list=4096 \
  ingress"

    echo "Ingress network recreated with subnet: $NEW_SUBNET"
else
    echo "Ingress network subnet is not 10.0.0.0/24. No action taken."
fi

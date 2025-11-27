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

set -Eeuo pipefail

set +x

if [ "$#" -lt 1 ]; then
  echo "Usage: $0 <ip address first swarm node>" >&2
  exit 1
fi
source ./utils.sh

# Define the desired subnet and gateway
NEW_SUBNET="192.168.44.0/24"
NEW_GATEWAY="192.168.44.1"
INGRESS_NETWORK_NAME=ingress

SSH_USER=$LOGNAME

# $1 is the hostname or IP address for the swarm manager
swarm_manager_addr="$1"

# Create a script to run on the remote VM
# This script will be executed via SSH on the remote host
remote_script=$(cat <<'REMOTE_SCRIPT_EOF'
set -Eeuo pipefail

SWARM_MANAGER_ADDR="$1"
NEW_SUBNET="$2"
NEW_GATEWAY="$3"
INGRESS_NETWORK_NAME="$4"

# Check if swarm is inactive and initialize if needed
if docker info --format '{{.Swarm.LocalNodeState}}' | grep -q "inactive"; then
  # Initialize Swarm with all nodes as managers and workers
  WORKER_TOKEN=$(docker swarm init --advertise-addr "$SWARM_MANAGER_ADDR" --listen-addr "$SWARM_MANAGER_ADDR":2377 | grep -o "SWMTKN-[^ ]*")
  echo "$WORKER_TOKEN" > worker.token
fi

# Get manager token
MANAGER_TOKEN=$(docker swarm join-token manager | grep -o "SWMTKN-[^ ]*")
echo "$MANAGER_TOKEN" > manager.token

# Get the current subnet of the ingress network
CURRENT_SUBNET=$(docker network inspect -f '{{range .IPAM.Config}}{{.Subnet}}{{end}}' ingress)
echo "$CURRENT_SUBNET"

# Check if the current subnet matches the desired subnet
if [ "$CURRENT_SUBNET" == "10.0.0.0/24" ]; then

  if docker network inspect "$INGRESS_NETWORK_NAME" > /dev/null 2>&1; then
    # Remove the existing ingress network
    echo "Remove ingress network"
    echo y | docker network rm ingress
  fi

  del_count=0
  while [ $del_count -lt 10 ]; do
    if ! docker network inspect "$INGRESS_NETWORK_NAME" > /dev/null 2>&1; then
      echo "ingress network deleted."
      break
    fi
    echo "Still waiting for delete ingress network.."
    sleep 2
      del_count=$((del_count+1))
  done

  if [ $del_count -eq 10 ]; then
    echo "Delete ingress network timed out."
    exit 1
  fi

  echo "Create ingress network"
  # Create a new ingress network with the desired subnet
  docker network create \
  --driver overlay \
  --ingress \
  --subnet="$NEW_SUBNET" \
  --gateway="$NEW_GATEWAY" \
  --opt com.docker.network.driver.overlay.vxlanid_list=4096 \
  ingress

    echo "Ingress network recreated with subnet: $NEW_SUBNET"
else
    echo "Ingress network subnet is not 10.0.0.0/24. No action taken."
fi
REMOTE_SCRIPT_EOF
)

# Execute the script on the remote VM
utils_ssh "$SSH_USER@$swarm_manager_addr" "bash -s $swarm_manager_addr $NEW_SUBNET $NEW_GATEWAY $INGRESS_NETWORK_NAME" <<< "$remote_script"

# Fetch the manager token from the remote VM and save it locally
MANAGER_TOKEN=$(utils_ssh "$SSH_USER@$swarm_manager_addr" "cat manager.token")
echo "$MANAGER_TOKEN" > manager.token

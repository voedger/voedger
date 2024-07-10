#!/usr/bin/env bash
#
# Copyright (c) 2023 Sigma-Soft, Ltd.
# @author Aleksei Ponomarev
#
# Add node to Swarm. Find node id with dedicated ip. If node id not found - join node to swarm cluster.
# Token, stored in 'manager.token' file used for join node.

set -euo pipefail

set +x

if [ "$#" -lt 2 ]; then
  echo "Usage: $0 <ip address first swarm node> <...>" >&2
  exit 1
fi

source ./utils.sh

SSH_USER=$LOGNAME
MANAGER=$1

JOIN_TOKEN=$(cat ./manager.token)

shift 
# Add remaining nodes as managers and workers
while [ $# -gt 0 ]; do

ip=$(getent hosts "$1" | awk '{print $1}')

# Get the ID of the node with the specified IP address
node_id=$(utils_ssh "$SSH_USER@$MANAGER" "docker node ls --format '{{.ID}}' | while read id; do docker node inspect --format '{{.Status.Addr}} {{.ID}}' \$id; done | grep $ip | awk '{print \$2}'")
  if [[ -n "$node_id" ]]; then
    echo "Host is already a member of Docker Swarm cluster."
  else 
    echo "Join node to Docker Swarm..."
    utils_ssh "$SSH_USER@$1" "docker swarm join --token $JOIN_TOKEN --listen-addr $ip:2377 $MANAGER:2377"

    # Wait until the node appears in the swarm node list
    echo "Waiting for the node to join the Docker Swarm cluster..."
    timeout=60  # Timeout in seconds
    interval=5  # Interval between checks in seconds
    while (( timeout > 0 )); do
      node_id=$(utils_ssh "$SSH_USER@$MANAGER" "docker node ls --format '{{.ID}}' | while read id; do docker node inspect --format '{{.Status.Addr}} {{.ID}}' \$id; done | grep $ip | awk '{print \$2}'")
      if [[ -n "$node_id" ]]; then
        echo "Node successfully joined the Docker Swarm cluster."
        break
      fi
      sleep $interval
      ((timeout -= interval))
    done

    if (( timeout <= 0 )); then
      echo "Failed to join the Docker Swarm cluster within the timeout period." >&2
      exit 1
    fi

  fi

shift

done

set +x

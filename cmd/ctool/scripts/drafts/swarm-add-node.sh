#!/usr/bin/env bash
#
# Copyright (c) 2023 Sigma-Soft, Ltd.
# @author Aleksei Ponomarev
#
# Add node to Swarm. Find node id with dedicated ip. If node id not found - join node to swarm cluster.
# Token, stored in 'manager.token' file used for join node.

set -Eeuo pipefail

set +x

if [ "$#" -lt 2 ]; then
  echo "Usage: $0 <manager ip address> <node ip address> [<node ip address> ...]" >&2
  exit 1
fi

source ./utils.sh

SSH_USER=$LOGNAME
MANAGER=$1

JOIN_TOKEN=$(cat ./manager.token)

shift
# Add remaining nodes as managers and workers
while [ $# -gt 0 ]; do

# $1 is now the IP address of the node to add
ip="$1"

# Get the ID of the node with the specified IP address
node_id=$(utils_ssh "$SSH_USER@$MANAGER" "docker node ls --format '{{.ID}}' | while read id; do docker node inspect --format '{{.Status.Addr}} {{.ID}}' \$id; done | grep $ip | awk '{print \$2}'")
  if [[ -n "$node_id" ]]; then
    echo "Host is already a member of Docker Swarm cluster."
  else
    echo "Join node to Docker Swarm..."
    # Attempt to join node to Docker Swarm
    join_output=$(utils_ssh "$SSH_USER@$ip" "docker swarm join --token $JOIN_TOKEN --listen-addr $ip:2377 $MANAGER:2377" 2>&1) || join_error=$?
      if [[ -n "${join_error-}" ]]; then
        if [[ "$join_output" == *"This node is already part of a swarm"* ]]; then
          echo "Node is already part of a swarm, leaving current swarm..."
          utils_ssh "$SSH_USER@$1" "docker swarm leave --force"
          echo "Rejoining node to Docker Swarm..."
          utils_ssh "$SSH_USER@$1" "docker swarm join --token $JOIN_TOKEN --listen-addr $ip:2377 $MANAGER:2377"
        else
          echo "Failed to join node to Docker Swarm: $join_output"
          exit 1
        fi
      fi
  fi

shift

done

set +x

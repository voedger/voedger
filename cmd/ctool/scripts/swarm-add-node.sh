#!/usr/bin/env bash
#
# Copyright (c) 2023 Sigma-Soft, Ltd.
# @author Aleksei Ponomarev

set -euo pipefail

set +x

if [ "$#" -lt 2 ]; then
  echo "Usage: $0 <ip address first swarm node> <...>" >&2
  exit 1
fi

SSH_USER=$LOGNAME
SSH_OPTIONS='-o UserKnownHostsFile=/dev/null -o StrictHostKeyChecking=no'
MANAGER=$1

JOIN_TOKEN=$(cat ./manager.token)

shift 
# Add remaining nodes as managers and workers
while [ $# -gt 0 ]; do

# Get the ID of the node with the specified IP address
node_id=$(ssh $SSH_OPTIONS $SSH_USER@$MANAGER "docker node ls --format '{{.ID}}' | while read id; do docker node inspect --format '{{.Status.Addr}} {{.ID}}' \$id; done | grep $1 | awk '{print \$2}'")
  if [[ -n "$node_id" ]]; then
    echo "Host is already a member of Docker Swarm cluster."
  else 
    echo "Join node to Docker Swarm..."
    ssh $SSH_OPTIONS $SSH_USER@$1 "docker swarm join --token $JOIN_TOKEN --listen-addr $1:2377 $MANAGER:2377"
  fi

shift

done

set +x

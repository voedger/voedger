#!/usr/bin/env bash
#
# Copyright (c) 2023 Sigma-Soft, Ltd.
# @author Aleksei Ponomarev
#
# Delete node from Swarm cluster.
#    - find node id by ip address
#    - Since manager cannot be removed from swarm,
#        - first, manager demote to worker
#        - then sequentally removed from swarm

set -Eeuo pipefail

set +x

if [ "$#" -lt 2 ]; then
  echo "Usage: $0 <swarm manager> <removing node> <...>" >&2
  exit 1
fi

source ./utils.sh

SSH_USER=$LOGNAME
MANAGER=$1

JOIN_TOKEN=$(cat ./manager.token)

shift
# Add remaining nodes as managers and workers
while [ $# -gt 0 ]; do

# Get the ID of the node with the specified IP address
node_id=$(utils_ssh "$SSH_USER@$MANAGER" "docker node ls --format '{{.ID}}' | while read id; do docker node inspect --format '{{.Status.Addr}} {{.ID}}' \$id; done | grep $1 | awk '{print \$2}'")
  if [[ -n "$node_id" ]]; then
    utils_ssh "$SSH_USER@$MANAGER" "docker node demote $node_id && docker node rm -f $node_id"
  fi

shift

done

set +x

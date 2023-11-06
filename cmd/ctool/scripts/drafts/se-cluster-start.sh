#!/usr/bin/env bash
#
# Copyright (c) 2023 Sigma-Soft, Ltd.
# @author Aleksei Ponomarev
#
# Deploy se cluster
#   - add labels to swarm nodes
#   - deploy se stack

set -euo pipefail

set -x

SSH_USER=$LOGNAME


if [[ $# -ne 2 ]]; then
  echo "Usage: $0 <node1> <node2>"
  exit 1
fi

source ./utils.sh

MANAGER=$1

# Add remaining nodes to swarm cluster
# while [ $# -gt 0 ]; do
#   ./swarm-set-label.sh $MANAGER $1 "se" "se"
#   shift
# done

# Start db cluster
cat ./docker-compose-se.yml | utils_ssh "$SSH_USER@$MANAGER" 'cat > ~/docker-compose-se.yml'

utils_ssh "$SSH_USER@$MANAGER" "docker stack deploy --compose-file ~/docker-compose-se.yml SEDockerStack"


set +x

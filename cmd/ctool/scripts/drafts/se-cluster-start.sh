#!/usr/bin/env bash
#
# Copyright (c) 2023 Sigma-Soft, Ltd.
# @author Aleksei Ponomarev
#
# Deploy se cluster
#   - add labels to swarm nodes
#   - deploy se stack

set -Eeuo pipefail

set -x

SSH_USER=$LOGNAME


if [[ $# -ne 2 ]]; then
  echo "Usage: $0 <node1> <node2>"
  exit 1
fi

source ./utils.sh

MANAGER=$1

# Replace placeholder with env var values and start db cluster
envsubst < ./docker-compose-se.yml | utils_ssh "$SSH_USER@$MANAGER" 'cat > ~/docker-compose-se.yml'

utils_ssh "$SSH_USER@$MANAGER" "docker stack deploy --compose-file ~/docker-compose-se.yml SEDockerStack"


set +x

#!/usr/bin/env bash
#
# Copyright (c) 2023 Sigma-Soft, Ltd.
# @author Aleksei Ponomarev

set -euo pipefail

set +x

if [ "$#" -lt 1 ]; then
  echo "Usage: $0 <ip address first swarm node>" >&2
  exit 1
fi

SSH_USER=$LOGNAME
SSH_OPTIONS='-o UserKnownHostsFile=/dev/null -o StrictHostKeyChecking=no'

if [[ $(ssh $SSH_OPTIONS $SSH_USER@$1 docker info --format '{{.Swarm.LocalNodeState}}') == "inactive" ]]; then
  # Initialize Swarm with all nodes as managers and workers
  WORKER_TOKEN=$(ssh $SSH_OPTIONS $SSH_USER@$1 docker swarm init --advertise-addr $1 --listen-addr $1:2377 | grep -oP "SWMTKN-\S+")
  echo $WORKER_TOKEN > worker.token
fi

MANAGER_TOKEN=$(ssh $SSH_OPTIONS $SSH_USER@$1 docker swarm join-token manager | grep -oP "SWMTKN-\S+")
echo $MANAGER_TOKEN > manager.token

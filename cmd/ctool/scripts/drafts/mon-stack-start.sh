#!/usr/bin/env bash
#
# Copyright (c) 2023 Sigma-Soft, Ltd.
# @author Aleksei Ponomarev
#
# Create docker-compose.yml for scylla stack and deploy

set -euo pipefail

set -x

if [[ $# -ne 1 ]]; then
  echo "Usage: $0 <swarm manager node>"
  exit 1
fi

SSH_USER=$LOGNAME
SSH_OPTIONS='-o UserKnownHostsFile=/dev/null -o StrictHostKeyChecking=no -o LogLevel=ERROR'

ssh $SSH_OPTIONS $SSH_USER@$1 "docker stack deploy --compose-file ~/docker-compose-mon.yml MonDockerStack"

set +x

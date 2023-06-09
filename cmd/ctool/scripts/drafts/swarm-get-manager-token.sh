#!/usr/bin/env bash
#
# Copyright (c) 2023 Sigma-Soft, Ltd.
# @author Aleksei Ponomarev
#
#    - get token for add managers
#    - store token for manangers to 'manager.token' file

set -euo pipefail

set +x

if [ "$#" -lt 1 ]; then
  echo "Usage: $0 <ip address first swarm node>" >&2
  exit 1
fi

SSH_USER=$LOGNAME
SSH_OPTIONS='-o UserKnownHostsFile=/dev/null -o StrictHostKeyChecking=no -o LogLevel=ERROR'

MANAGER_TOKEN=$(ssh $SSH_OPTIONS $SSH_USER@$1 docker swarm join-token --rotate manager | grep -oP "SWMTKN-\S+")
echo $MANAGER_TOKEN > manager.token

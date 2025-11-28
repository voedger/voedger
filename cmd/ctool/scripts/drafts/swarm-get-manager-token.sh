#!/usr/bin/env bash
#
# Copyright (c) 2023 Sigma-Soft, Ltd.
# @author Aleksei Ponomarev
#
#    - get token for add managers
#    - store token for manangers to 'manager.token' file

set -Eeuo pipefail

set +x

source ./utils.sh

if [ "$#" -lt 1 ]; then
  echo "Usage: $0 <ip address first swarm node>" >&2
  exit 1
fi

SSH_USER=$LOGNAME

MANAGER_TOKEN=$(utils_ssh "$SSH_USER@$1" docker swarm join-token --rotate manager | grep -oP "SWMTKN-\S+")
echo "$MANAGER_TOKEN" > manager.token

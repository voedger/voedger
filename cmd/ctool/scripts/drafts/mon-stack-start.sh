#!/usr/bin/env bash
#
# Copyright (c) 2023 Sigma-Soft, Ltd.
# @author Aleksei Ponomarev
#
# Create docker-compose.yml for scylla stack and deploy

set -euo pipefail

set -x

if [[ $# -ne 2 ]]; then
  echo "Usage: $0 <app-node-1> <app-node-2>"
  exit 1
fi

source ./utils.sh

SSH_USER=$LOGNAME

AppNode1=$1
AppNode2=$2

cat ./docker-compose-mon.yml | \
    sed "s/{{.AppNode1}}/app-node-1/g; s/{{.AppNode2}}/app-node-2/g" \
    | utils_ssh "$SSH_USER@$AppNode1" 'cat > ~/docker-compose-mon.yml'

utils_ssh "$SSH_USER@$AppNode1" "docker stack deploy --compose-file ~/docker-compose-mon.yml MonDockerStack"

set +x

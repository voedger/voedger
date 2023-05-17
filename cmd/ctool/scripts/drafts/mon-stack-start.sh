#!/usr/bin/env bash
#
# Copyright (c) 2023 Sigma-Soft, Ltd.
# @author Aleksei Ponomarev
#
# Create docker-compose.yml for scylla stack and deploy

set -euo pipefail

set -x

if [[ $# -ne 2 ]]; then
  echo "Usage: $0 <se1> <se2>" 
  exit 1
fi

SSH_USER=$LOGNAME
SSH_OPTIONS='-o UserKnownHostsFile=/dev/null -o StrictHostKeyChecking=no -o LogLevel=ERROR'

se1=$1
se2=$2

cat ./docker-compose-mon.yml | \
    sed "s/{se1}/$se1/g; s/{se2}/$se2/g" \
    | ssh $SSH_OPTIONS $SSH_USER@$se1 'cat > ~/docker-compose-mon.yml'

ssh $SSH_OPTIONS $SSH_USER@$se1 "docker stack deploy --compose-file ~/docker-compose-mon.yml MonDockerStack"

set +x

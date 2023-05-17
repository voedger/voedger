#!/usr/bin/env bash
#
# Copyright (c) 2023 Sigma-Soft, Ltd.
# @author Aleksei Ponomarev
#
# Replace scylla node in cluster 
#   - Install docker on new hardware
#   - Add new node to swarm
#   - Prepare node to scylla service
#   - Prepare node to bootstrap
#   - Remove lost node from swarm cluster
#   - Prepare updated scylla stack
#   - deploy stack
#   - add label to new node, to place service 

set -euo pipefail

set -x

if [[ $# -ne 3 ]]; then
  echo "Usage: $0 <lost node> <new node> <swarm manager>"
  exit 1
fi

SSH_USER=$LOGNAME
SSH_OPTIONS='-o UserKnownHostsFile=/dev/null -o StrictHostKeyChecking=no -o LogLevel=ERROR'

MANAGER=$3

./docker-install.sh $2

./swarm-add-node.sh $MANAGER $2

./db-node-prepare.sh $2

./db-bootstrap-prepare.sh $1 $2

./swarm-rm-node.sh $MANAGER $1

service_label=$(./db-stack-update.sh $1 $2 | tail -n 1)

cat ./docker-compose.yml | ssh $SSH_OPTIONS $SSH_USER@$2 'cat > ~/docker-compose.yml'

ssh $SSH_OPTIONS $SSH_USER@$2 "docker stack deploy --compose-file ~/docker-compose.yml DBMSDockerStack"

./swarm-set-label.sh $MANAGER $2 $service_label
set +x

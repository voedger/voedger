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
REPLACED_NODE_NAME=$(getent hosts "$2" | awk '{print $2}')

wait_for_scylla() {
    local ip_address=$1
    echo "Working with $ip_address"
    local count=0

    while [ $count -lt 100 ]; do
        if [ "$(ssh "$SSH_OPTIONS" "$SSH_USER"@"$ip_address" docker exec '$(docker ps -qf name=scylla)' nodetool status | grep -c '^UN\s')" -eq 3 ]; then
            echo "Scylla initialization success"
            return 0
        fi
        echo "Still waiting for Scylla initialization.."
        sleep 5
        count=$((count+1))
    done
    if [ $count -eq 100 ]; then
        echo "Scylla initialization timed out."
        return 1
    fi
}

./docker-install.sh $2

./swarm-add-node.sh $MANAGER $2

./db-node-prepare.sh $2

./db-bootstrap-prepare.sh $1 $2

./swarm-rm-node.sh $MANAGER $1

service_label=$(./db-stack-update.sh "$REPLACED_NODE_NAME" | tail -n 1)

cat ./docker-compose.yml | ssh $SSH_OPTIONS $SSH_USER@$2 'cat > ~/docker-compose.yml'

ssh $SSH_OPTIONS $SSH_USER@$2 "docker stack deploy --compose-file ~/docker-compose.yml DBDockerStack"

./swarm-set-label.sh $MANAGER $2 "type" $service_label

wait_for_scylla "$2"

set +x

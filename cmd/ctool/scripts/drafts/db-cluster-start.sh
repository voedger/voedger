#!/usr/bin/env bash
#
# Copyright (c) 2023 Sigma-Soft, Ltd.
# @author Aleksei Ponomarev
#
# Create docker-compose.yml for scylla stack and deploy

set -euo pipefail

set -x

if [[ $# -ne 3 ]]; then
  echo "Usage: $0 <DBNode1> <DBNode2> <DBNode3>"
  exit 1
fi

SSH_USER=$LOGNAME
SSH_OPTIONS='-o UserKnownHostsFile=/dev/null -o StrictHostKeyChecking=no -o LogLevel=ERROR'

DBNode1=$1
DBNode2=$2
DBNode3=$3

# Replace the template values in the YAML file with the arguments (scylla nodes ip addresses)
# and store as prod compose file for start swarm services
cat docker-compose-template.yml | \
    sed "s/{{.DBNode1})/$DBNode1/g; s/{{.DBNode2}}/$DBNode2/g; s/{{.DBNode3}}/$DBNode3/g" \
    > ./docker-compose.yml

cat ./docker-compose.yml | ssh $SSH_OPTIONS $SSH_USER@$1 'cat > ~/docker-compose.yml'

ssh $SSH_OPTIONS $SSH_USER@$1 "docker stack deploy --compose-file ~/docker-compose.yml DBDockerStack"

set +x

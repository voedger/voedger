#!/usr/bin/env bash
#
# Copyright (c) 2023 Sigma-Soft, Ltd.
# @author Aleksei Ponomarev
#
# Create docker-compose.yml for scylla stack and deploy

set -euo pipefail

set -x

if [[ $# -ne 3 ]]; then
  echo "Usage: $0 <scylla1> <scylla2> <scylla3>"
  exit 1
fi

SSH_USER=$LOGNAME
SSH_OPTIONS='-o UserKnownHostsFile=/dev/null -o StrictHostKeyChecking=no'

scylla1=$1
scylla2=$2
scylla3=$3

# Replace the template values in the YAML file with the arguments (scylla nodes ip addresses)
# and store as prod compose file for start swarm services
cat docker-compose-template.yml | \
    sed "s/{scylla1}/$scylla1/g; s/{scylla2}/$scylla2/g; s/{scylla3}/$scylla3/g" \
    > ./docker-compose.yml

cat ./docker-compose.yml | ssh $SSH_OPTIONS $SSH_USER@$1 'cat > ~/docker-compose.yml'

ssh $SSH_OPTIONS $SSH_USER@$1 "docker stack deploy --compose-file ~/docker-compose.yml scylla"

set +x

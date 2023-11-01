#!/usr/bin/env bash
#
# Copyright (c) 2023 Sigma-Soft, Ltd.
# @author Aleksei Ponomarev
#
# Create docker-compose.yml for scylla stack

set -euo pipefail

set -x

if [[ $# -ne 4 ]]; then
  echo "Usage: $0 <scylla1> <scylla2> <scylla3> <dev mode>"
  exit 1
fi

SSH_USER=$LOGNAME
SSH_OPTIONS='-o UserKnownHostsFile=/dev/null -o StrictHostKeyChecking=no -o LogLevel=ERROR'

local SERVICES=$(yq eval '.services | keys | map(select(test("^scylla"))) | .[]' docker-compose-template.yml)
  for SERVICE in $SERVICES; do
    yq eval '.services."'$SERVICE'".command |= sub("--developer-mode [0-9]+", "--developer-mode '"$DEVELOPER_MODE_VALUE"'")' -i docker-compose-template.yml
  done

yq e '.services |= with_entries(.value.command |= sub("--developer-mode [0-9]+", "--developer-mode '"$DEVELOPER_MODE_VALUE"'"))' -i docker-compose-template.yml

# Replace the template values in the YAML file with the arguments (scylla nodes ip addresses)
# and store as prod compose file for start swarm services
cat docker-compose-template.yml | \
    sed "s/{{\.DBNode1}}/$1/g; s/{{\.DBNode2}}/$2/g; s/{{\.DBNode3}}/$3/g" \
    > ./docker-compose.yml



set +x

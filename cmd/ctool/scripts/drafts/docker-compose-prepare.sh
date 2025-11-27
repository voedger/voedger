#!/usr/bin/env bash
#
# Copyright (c) 2023 Sigma-Soft, Ltd.
# @author Aleksei Ponomarev
#
# Create docker-compose.yml for scylla stack

set -Eeuo pipefail

set -x

  if [[ $# -ne 4 ]]; then
    echo "Usage: $0 <scylla1> <scylla2> <scylla3> <dev mode>"
    exit 1
  fi

source ./utils.sh

SSH_USER=$LOGNAME

if [ "$4" != "1" ] && [ "$4" != "0" ]; then
  echo "Usage: $0 <scylla1> <scylla2> <scylla3> <dev mode: 1 | 0>"
  echo "dev mode should be 0 or 1"
  exit 1
fi

declare -A node_map

node_map["scylla1"]="$1"
node_map["scylla2"]="$2"
node_map["scylla3"]="$3"

# Function to convert names
convert_name() {
  local input_name="$1"
  local converted_name="${node_map[$input_name]}"
  if [ -n "$converted_name" ]; then
      echo "$converted_name"
  else
    echo "Error: Unknown name '$input_name'" >&2
    exit 1
  fi
}


#--io-setup 0
# io_properties.yaml
DEVELOPER_MODE=$4
echo "DEVELOPER_MODE=$DEVELOPER_MODE"

SERVICES=$(yq eval '.services | keys | map(select(test("^scylla"))) | .[]' docker-compose-template.yml)
  for SERVICE in $SERVICES; do
    yq eval '.services."'$SERVICE'".command |= sub("--developer-mode [0-9]+", "--developer-mode '"$DEVELOPER_MODE"'")' -i docker-compose-template.yml
    # node=$(convert_name "$SERVICE")
      # if ssh "$SSH_OPTIONS" "$SSH_USER"@"$node" "test -s ~/scylla.d/io_properties.yaml && echo 'io_properties.yaml exist, will skip scylla_io_setup'; exit \$? || echo 'io_properties.yaml not exist, scylla_io_setup will run'; exit \$?"; then
        # yq eval '.services."'$SERVICE'".command = (.services."'$SERVICE'".command | sub("--io-setup 1", "--io-setup 0"))' -i docker-compose-template.yml
      # else
        # yq eval '.services."'$SERVICE'".command = (.services."'$SERVICE'".command | sub("--io-setup 0", "--io-setup 1"))' -i docker-compose-template.yml
      # fi
  done

# Replace the template values in the YAML file with the arguments (scylla nodes ip addresses)
# and store as prod compose file for start swarm services
cat docker-compose-template.yml | \
    sed "s/{{\.DBNode1}}/$1/g; s/{{\.DBNode2}}/$2/g; s/{{\.DBNode3}}/$3/g" \
    > ./docker-compose.yml

set +x

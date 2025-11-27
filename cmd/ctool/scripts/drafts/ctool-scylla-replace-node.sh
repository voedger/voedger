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
#   - add node to back seed list and restart service
#   - rolling restart db cluster

set -Eeuo pipefail

set -x

if [[ $# -lt 3 ]]; then
  echo "Usage: $0 <lost node> <new node> <swarm manager> <datacenter>"
  exit 1
fi

source ./utils.sh

SSH_USER=$LOGNAME
STACK="DBDockerStack"

declare -A service_map

service_map["db-node-1"]="scylla1"
service_map["db-node-2"]="scylla2"
service_map["db-node-3"]="scylla3"

# Function to convert names
convert_name() {
  local input_name="$1"
  local converted_name="${service_map[$input_name]}"
  if [ -n "$converted_name" ]; then
      echo "$converted_name"
  else
    echo "Error: Unknown name '$input_name'" >&2
    exit 1
  fi
}

MANAGER=$3
if [ -n "${4+x}" ] && [ -n "$4" ]; then
  DC=$4
  else
    DC=""
fi

REPLACED_NODE_NAME=$(grep "$2" /etc/hosts | grep db-node | awk '{print $2}')
ssh-keyscan -p "$(utils_SSH_PORT)" -H "$REPLACED_NODE_NAME" >> ~/.ssh/known_hosts

# Function to check if Scylla server is up and listening
scylla_is_listen() {
  local SCYLLA_HOST="$1"
  local SCYLLA_PORT="$2"

  if nc -zvw3 "$SCYLLA_HOST" "$SCYLLA_PORT"; then
    return 0  # Server is up and listening
  else
    return 1  # Server is not reachable
  fi
}

# Function to wait for ready scylla cluster
scylla_wait() {
  local ip_address=$1
  echo "Working with $ip_address"
  local count=0
  local listen_attempts=0
  local max_attempts=300
  local timeout=10

  sleep "$timeout"

  while [ $count -lt $max_attempts ]; do
      if [ "$(utils_ssh "$SSH_USER@$ip_address" docker exec '$(docker ps -qf name=scylla --filter status=running)' nodetool status | grep -c '^UN\s')" -eq 3 ]; then
          echo "Scylla cluster initialization success. Check scylla is listening on interface."

          while [ $listen_attempts -lt $max_attempts ]; do
              ((++listen_attempts))
              echo "Attempt $listen_attempts: Checking Scylla server..."
              if scylla_is_listen "$ip_address" 9042; then
                  echo "Scylla server is up and ready."
                  return 0
              else
                  echo "Scylla server is not yet ready. Retrying in $timeout seconds..."
                  sleep "$timeout"
              fi
          done

          if [ $listen_attempts -eq $max_attempts ]; then
            echo "Max attempts reached. Scylla server is still not ready."
            return 1
          fi
      fi
      echo "Still waiting for Scylla initialization.."
      sleep $timeout
      count=$((count+1))
  done
  if [ $count -eq $max_attempts ]; then
      echo "Scylla initialization timed out."
      return 1
  fi
}

wait_for_scylla() {
  local ip_address=$1
  echo "Working with $ip_address"
  local count=0
  sleep 10
  while [ $count -lt 300 ]; do
    if [ $(utils_ssh "$SSH_USER@$ip_address" "docker exec \$(docker ps -qf name=scylla --filter status=running) nodetool status | grep -c '^UN\s'") -eq 3 ]; then
      echo "Scylla initialization success"
      return 0
    fi
    echo "Still waiting for Scylla initialization.."
    sleep 10
    count=$((count+1))
  done
  if [ $count -eq 300 ]; then
    echo "Scylla initialization timed out."
    return 1
  fi
}

#./docker-install.sh "$2"
./swarm-add-node.sh "$MANAGER" "$2"
./swarm-rm-node.sh "$MANAGER" "$1"
./db-node-prepare.sh "$2" "$DC"
./db-bootstrap-prepare.sh "$1" "$2"

seed_list() {
  local node=$1
  local operation=$2

  service_label=$(./db-stack-update.sh "$node" "$operation" | tail -n 1)
  < ./docker-compose.yml utils_ssh "$SSH_USER@$node" 'cat > ~/docker-compose.yml'
  utils_ssh "$SSH_USER@$node" "docker stack deploy --compose-file ~/docker-compose.yml DBDockerStack"
  ./swarm-set-label.sh "$MANAGER" "$node" "$service_label" "true"
}

echo "Remove dead node from seed list and start db instance on new hardware."
seed_list "$REPLACED_NODE_NAME" remove

scylla_wait "$REPLACED_NODE_NAME"

echo "Bootstrap complete. Cleanup scylla config..."
./db-bootstrap-end.sh "$2"

REPLACED_SERVICE=$(convert_name "$REPLACED_NODE_NAME")
echo "Add new live node to back to seed list and restart."
seed_list "$REPLACED_NODE_NAME" add

scylla_wait "$REPLACED_NODE_NAME"

utils_ssh "$SSH_USER@$REPLACED_NODE_NAME" "docker exec \$(docker ps -qf name=scylla --filter status=running) nodetool repair -full"

db_rolling_restart() {
  local compose_file="$1"
  local services=()

  mapfile -t services < <(yq eval '.services | keys | .[]' "$compose_file" | grep -v "$REPLACED_SERVICE")

  for service in "${services[@]}"; do
    echo "Restart service: ${STACK}_${service}"
    docker service update --force "$STACK"_"$service"
    wait_for_scylla "$REPLACED_NODE_NAME"
  done
}
#echo "Rolling restart db cluster..."
#db_rolling_restart ./docker-compose.yml

set +x

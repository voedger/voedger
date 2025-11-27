#!/usr/bin/env bash
# Copyright (c) 2024 Sigma-Soft, Ltd.
# @author Aleksei Ponomarev
#
# Backup scylla nodes
# Usage: ./restorenode.sh <Target folder with backup> <Path to ssh key> <Node1> <Node2> <Node3> <Node...>
# Example: ./restorenode.sh /mnt/backup/test-cluster-20240205 /home/ubuntu/.ssh/id_rsa 10.0.0.13 10.0.0.14 10.0.0.15
#
# Operations:
# - Prepare
#    - get container id
#    - get users keyspaces available in backup
#    - recreate keyspace with tables
#    - create mapping between new table id and backup data
# - Restore
#    - stop db services
#    - clean up commitlog
#    - load data from backup
#    - start db services
# - Post recovery steps
#    - wait for scylla initialization
#    - run nodetool repair

set -Eeuo pipefail

set +x

if [[ $# -lt 2 ]]; then
  echo "Usage: $0 <Folder with backup> <Node1> <Node2> <Node3> <Node...>"
  exit 1
fi

source ./utils.sh
source ./utils-cql.sh

readonly targetFolder=$1
readonly containerName="scylla"
readonly nodeDataDir="/var/lib/scylla"


descUserKeyspaces() {
    local -n input_array=$1
    local -a filtered_array=()

    for element in "${input_array[@]}"; do
        if [[ $element != system* ]]; then
            filtered_array+=("$element")
        fi
    done

    input_array=("${filtered_array[@]}")
}

function hasKeyspace() {
    local keyspace=$1
    local container=$2
    local -a keyspaces

    # shellcheck disable=SC2034
    keyspaces=$(descKeyspaces "$container")
    descUserKeyspaces "keyspaces"

    if printf "%s\n" "${keyspaces[@]}" | grep -q "$keyspace"; then
        return 0
    fi
    echo "Keyspace $keyspace not found"
    return 1
}

function isSwarmEnabled() {
    local swarm_status
    swarm_status=$(utils_ssh "$LOGNAME@$node" docker info --format '{{.Swarm.LocalNodeState}}')

    if [ "$swarm_status" == "active" ]; then
        echo 0  # Docker Swarm is enabled
    else
        echo 1  # Docker Swarm is not enabled
    fi
}

function serviceCtl() {
    local container="$1"

    case "$2" in
        stop)
            if [ "$(isSwarmEnabled)" -eq 1 ]; then
                echo "Docker Swarm is not enabled."
                utils_ssh "$LOGNAME@$node" docker exec -i "$container" nodetool drain
                utils_ssh "$LOGNAME@$node" docker stop "$container"
            else
                echo "Docker Swarm enabled. Scale to zero..."
                utils_ssh "$LOGNAME@$node" docker service scale DBDockerStack_scylla1=0
                utils_ssh "$LOGNAME@$node" docker service scale DBDockerStack_scylla2=0
                utils_ssh "$LOGNAME@$node" docker service scale DBDockerStack_scylla3=0
            fi

            echo "Service stopped."
            ;;
        start)
           # Remove the signal file if it exists
            if utils_ssh "$LOGNAME@$node" "[ -e $signalFilePath ]"; then
                echo $?  # 0 - signal file exists
                utils_ssh "$LOGNAME@$node" rm "$signalFilePath"
                echo "Signal file removed from: $signalFilePath"
            else
                echo "Signal file not found at: $signalFilePath"
            fi
            if [ "$(isSwarmEnabled)" -eq 1 ]; then
                utils_ssh "$LOGNAME@$node" docker start "$container"
            else
                utils_ssh "$LOGNAME@$node" docker service scale DBDockerStack_scylla1=1
                utils_ssh "$LOGNAME@$node" docker service scale DBDockerStack_scylla2=1
                utils_ssh "$LOGNAME@$node" docker service scale DBDockerStack_scylla3=1
            fi
            ;;
        *)
            echo "Invalid operation. Use 'stop' or 'start'."
            ;;
    esac
}

function truncateKeyspace() {
    local node=$1
    local keyspace=$2
    local container=$3
        echo "DROP KEYSPACE $keyspace on node $node with container $container"
        utils_ssh "$LOGNAME@$node" docker exec -i "$container" "cqlsh -u cassandra -p cassandra -e \"DROP KEYSPACE $keyspace\""
    return $?
}

function restoreKeyspace() {
  echo 0
}

tableLoad() {
    local keyspace=$1
    local key="$2"
    local value="$3"
    utils_ssh "$LOGNAME@$node" sudo rm -rf "$nodeDataDir/commitlog/*"
    utils_ssh "$LOGNAME@$node" sudo find "$nodeDataDir/data/$keyspace/$key-${value//-/}" -type f -exec rm {} +

    max_depth=$(utils_ssh "$LOGNAME@$node" tar -tf "$targetFolder/$keyspace/data.tar.gz" | grep '/schema.cql$' | awk -F/ '{print NF-1}' | sort -rn | head -n1)
    echo "Archive max depth: $max_depth"
    if [[ -n $max_depth ]]; then
        utils_ssh "$LOGNAME@$node" sudo tar -xzvf "$targetFolder/$keyspace/data.tar.gz" --strip-components="$max_depth" --directory="$nodeDataDir/data/$keyspace/$key-${value//-/}" --wildcards "*$key"-*/snapshots/*/*
    else
        echo "Cannot calculate backup archive depth. No trusting to archieve. Exit..."
        exit 1
    fi

    echo "Table: $key, Id: $value loaded"
}


function prepare() {
    local node=$1
    local keyspace=$2
    local container
    if container=$(getContainer "$containerName"); then
        echo "Container id for $containerName: $container"
    else
        echo "Failed to get Container id for $containerName"
        return 1
    fi

    if ! hasKeyspace "$keyspace" "$container"; then
        echo "Keyspace $keyspace no exists. Create..."
        utils_ssh "$LOGNAME@$node" docker exec -i "$container" "cqlsh -u cassandra -p cassandra < \"$targetFolder/$keyspace/schema.cql\""
    else
        if truncateKeyspace "$node" "$keyspace" "$container"; then
            echo "Keyspace $2 truncated"
            utils_ssh "$LOGNAME@$node" docker exec -i "$container" "cqlsh -u cassandra -p cassandra < \"$targetFolder/$keyspace/schema.cql\""
        else
            echo "Error truncating keyspace $keyspace"
            return 1
        fi
    fi
}

function restore() {
    local node=$1
    local json_table=$2
    local container
    local keyspaces
 #   local t=$3

    keyspaces=$(jq -r '.keyspaces[].name' <<< "$json_table")

    while read -r ks; do
        ks=$(echo "$ks" | xargs)
        echo "Processing keyspace: $ks"

        echo "$json_tables" | jq -r --arg key "$ks" '(.keyspaces[] | select(.name == $key) | .tables[] | "\(.table) \(.id)")' | while read -r key value; do
            tableLoad "$ks" "$key" "$value"
        done

    done <<< "$keyspaces"
}

scylla_is_listen() {
  local SCYLLA_HOST="$1"
  local SCYLLA_PORT="$2"

  if nc -zvw3 "$SCYLLA_HOST" "$SCYLLA_PORT"; then
    return 0  # Server is up and listening
  else
    return 1  # Server is not reachable
  fi
}

scylla_wait() {
local ip_address=$1
echo "Working with $ip_address"
local count=0
local listen_attempts=0
local max_attempts=90
local timeout=5

while [ $count -lt 90 ]; do
    if [ "$(utils_ssh "$LOGNAME"@"$ip_address" docker exec '$(docker ps -qf name=scylla)' nodetool status | grep -c '^UN\s')" -eq 3 ]; then
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

        if [ "$listen_attempts" -eq "$max_attempts" ]; then
            echo "Max attempts reached. Scylla server is still not ready."
            return 1
        fi
    fi
    echo "Still waiting for Scylla initialization.."
    sleep 5
    count=$((count+1))
done

if [ $count -eq 90 ]; then
    echo "Scylla initialization timed out."
    return 1
fi

}

shift 1

node="$1"
container=$(getContainer "$containerName")

if utils_ssh "$LOGNAME@$node" "[ -d \"$targetFolder\" ]"; then
    json_tables=$(jq -n -c -M '{keyspaces: []}')

    mapfile -t dirs < <(utils_ssh "$LOGNAME@$node" "find $targetFolder -mindepth 1 -maxdepth 1 -type d -exec basename {} \;")

    descUserKeyspaces "dirs"

    for dir in "${dirs[@]}"; do
        keyspace="$dir"
        echo "Found backup for keyspace: $keyspace"

        json_tables=$(jq -M --arg key "$keyspace" '.keyspaces += [{name: $key, tables: []}]' <<< "$json_tables")
        prepare "$node" "$keyspace"

        schema_tables=$(tables "$container" "$keyspace")
        while read -r key value; do
      			json_tables=$(jq -M --arg tbl "$key" --arg ids "$value" --arg key "$keyspace" '(.keyspaces[] | select(.name == $key) | .tables) += [{table: $tbl, id: $ids}]'  <<< "$json_tables")
            echo "Add table: $key with id: $value to keyspace: $keyspace schema"
        done < <(echo "$schema_tables" | jq -r 'to_entries[] | "\(.key) \(.value)"')

    done
    echo "$json_tables"
else
    echo "Folder $targetFolder does not exist."
    exit 1
fi

serviceCtl "$container" stop
while [ $# -gt 0 ]; do
    node="$1"
        restore "$node" "$json_tables"
        echo "Node: $node restored."
    shift
done
serviceCtl "$container" start

scylla_wait "$node"
utils_ssh "$LOGNAME"@"$node" docker exec '$(docker ps -qf name=scylla)' nodetool repair --full

exit 0


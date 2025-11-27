#!/usr/bin/env bash
# Copyright (c) 2024 Sigma-Soft, Ltd.
# @author Aleksei Ponomarev
#
# Restore scylla node from backup
# Usage: ./ce-restore-node.sh <Target folder with backup>
# Example: ./restorenode.sh /mnt/backup/
#
# Operations:
# - Prepare
#    - get container id
#    - get users keyspaces available in backup
#    - recreate keyspace with tables
#    - create mapping between new table id and backup data
# - Restore
#    - stop db service
#    - clean up commitlog
#    - load data from backup
#    - start db services
# - Post recovery steps
#    - wait for scylla initialization
#    - run nodetool repair

set -Eeuo pipefail

set +x

if [[ $# -lt 1 ]]; then
  echo "Usage: $0 <Folder with backup>"
  exit 1
fi

readonly targetFolder=$1
readonly containerName="scylla"
readonly nodeDataDir="/var/lib/scylla"
readonly signalFilePath="$HOME/ctool/.voedgerbackup"

getContainer() {
    containerID=$(docker ps -l -q -f "name=$containerName")
    if [ -z "$containerID" ]; then
        echo "Error getting container ID for $containerName"
        return 1
    fi

    trimmedID=$(echo "$containerID" | tr -d '[:space:]')

    if [ -z "$trimmedID" ]; then
        echo "Container ID not found for $containerName"
        return 1
    fi

    echo "$trimmedID"
}

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

tables() {
    local container="$1"
    local keyspace="$2"

    docker exec "$container" cqlsh -e "select table_name, id from system_schema.tables where keyspace_name='$keyspace'" | \
    grep -v '^$' | sed '/^Warning:/d' | tail -n +3 | head -n -1 | \
    jq -R -n '[inputs | split("|") | {(.[0] | gsub("^ +| +$";"")): (. [1] | gsub("^ +| +$";""))}] | add'
}


descKeyspaces() {
    containerID="$1"
    mapfile -t CQLout < <(docker exec "$containerID" cqlsh -e 'DESC KEYSPACES' | grep -v '^$' | sed '/^Warning:/d')

    # Create a new array to store trimmed and split keyspaces
    declare -a keyspaces

    # Process each line to split by space and trim elements
    for ((i = 0; i < ${#CQLout[@]}; i++)); do
        read -ra keyspaces_line <<< "${CQLout[i]}"
        trimmed_ks=""
        for ks in "${keyspaces_line[@]}"; do
            trimmed_ks+=$(echo "$ks" | tr -d '[:space:]')' '
        done
        # Remove trailing space
        keyspaces+=("${trimmed_ks%" "}")
    done

 echo "${keyspaces[@]}"
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

function serviceCtl() {
    local container="$1"

    case "$2" in
        stop)
            docker exec -i "$container" nodetool drain
            docker stop "$container"
            echo "Service stopped."
            ;;
        start)
           # Remove the signal file if it exists
            if [ -e "$signalFilePath" ]; then
                echo $?  # 0 - signal file exists
                rm "$signalFilePath"
                echo "Signal file removed from: $signalFilePath"
            else
                echo "Signal file not found at: $signalFilePath"
            fi
                docker start "$container"
            ;;
        *)
            echo "Invalid operation. Use 'stop' or 'start'."
            ;;
    esac
}

function truncateKeyspace() {
    local keyspace=$1
    local container=$2
        echo "DROP KEYSPACE $keyspace on node with container $container"
        docker exec -i "$container" cqlsh -u cassandra -p cassandra -e 'DROP KEYSPACE "'$keyspace'"'
    return $?
}

function restoreKeyspace() {
  echo 0
}

tableLoad() {
    local keyspace=$1
    local key="$2"
    local value="$3"
    sudo rm -rf "$nodeDataDir/commitlog/*"
    sudo find "$nodeDataDir/data/$keyspace/$key-${value//-/}" -type f -exec rm {} +
    max_depth=$(tar -tf "$targetFolder/$keyspace/data.tar.gz" | grep '/schema.cql$' | awk -F/ '{print NF-1}' | sort -rn | head -n1)
    echo "Archive max depth: $max_depth"
    if [[ -n $max_depth ]]; then
        sudo tar -xzvf "$targetFolder/$keyspace/data.tar.gz" --strip-components="$max_depth" --directory="$nodeDataDir/data/$keyspace/$key-${value//-/}" --wildcards "*$key"-*/snapshots/*/*
    else
        echo "Cannot calculate backup archive depth. No trusting to archieve. Exit..."
        exit 1
    fi
    echo "Table: $key, Id: $value loaded"
}


function prepare() {
    local keyspace=$1
    local container
    if container=$(getContainer "$containerName"); then
        echo "Container id for $containerName: $container"
    else
        echo "Failed to get Container id for $containerName"
        return 1
    fi

    if ! hasKeyspace "$keyspace" "$container"; then
        echo "Keyspace $keyspace no exists. Create..."
        docker exec -i "$container" cqlsh -u cassandra -p cassandra < "$targetFolder/$keyspace/schema.cql"
    else
        if truncateKeyspace "$keyspace" "$container"; then
            echo "Keyspace $1 truncated"
            docker exec -i "$container" cqlsh -u cassandra -p cassandra < "$targetFolder/$keyspace/schema.cql"
        else
            echo "Error truncating keyspace $keyspace"
            return 1
        fi
    fi
}

function restore() {
    local json_table=$1
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
    if [ "$(docker exec "$(docker ps -qf name=scylla)" nodetool status | grep -c '^UN\s')" -eq 1 ]; then
        echo "Scylla service initialization success. Check scylla is listening on interface."

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

container=$(getContainer "$containerName")

if  [ -d "$targetFolder" ]; then
    json_tables=$(jq -n -c -M '{keyspaces: []}')

    mapfile -t dirs < <(find "$targetFolder" -mindepth 1 -maxdepth 1 -type d -exec basename {} \;)

    descUserKeyspaces "dirs"

    for dir in "${dirs[@]}"; do
        keyspace="$dir"
        echo "Found backup for keyspace: $keyspace"

        json_tables=$(jq -M --arg key "$keyspace" '.keyspaces += [{name: $key, tables: []}]' <<< "$json_tables")
        prepare "$keyspace"

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
restore "$json_tables"
echo "Node: restored."
serviceCtl "$container" start

scylla_wait db-node-1
docker exec "$(docker ps -qf name=scylla)" nodetool repair --full

exit 0


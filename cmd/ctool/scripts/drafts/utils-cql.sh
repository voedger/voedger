#!/usr/bin/env bash
# Copyright (c) 2023 Sigma-Soft, Ltd.
# @author Aleksei Ponomarev

set -Eeuo pipefail

set +x

readonly signalFilePath="$HOME/ctool/.voedgerbackup"

getContainer() {
    if ! containerID=$(utils_ssh "$LOGNAME@$node" docker ps -l -q -f "name=$containerName"); then
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

tables() {
    local container="$1"
    local keyspace="$2"

    utils_ssh "$LOGNAME@$node" docker exec "$container" cqlsh -e "\"select table_name, id from system_schema.tables where keyspace_name='"$keyspace"'\"" | \
    grep -v '^$' | sed '/^Warning:/d' | tail -n +3 | head -n -1 | \
    jq -R -n '[inputs | split("|") | {(.[0] | gsub("^ +| +$";"")): (. [1] | gsub("^ +| +$";""))}] | add'
}

descKeyspaces() {
    containerID="$1"
    mapfile -t CQLout < <(
        utils_ssh "$LOGNAME@$node" \
        "docker exec $containerID cqlsh -e 'DESC KEYSPACES' | grep -v '^$' | sed '/^Warning:/d'"
    )

    if [ $? -ne 0 ]; then
        echo "Error executing 'DESC KEYSPACES' in container $containerID"
        return 1
    fi

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

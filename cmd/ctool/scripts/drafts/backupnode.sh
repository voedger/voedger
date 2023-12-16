#!/usr/bin/env bash
# Copyright (c) 2023 Sigma-Soft, Ltd.
# @author Aleksei Ponomarev
#
# Backup scylla node. Execute command over ssh
# Usage: ./backupnode.sh <Node> <Target folder> <Path to ssh key>
# Example: ./backupnode.sh 127.0.0.1 /tmp/backup /home/user/.ssh/id_rsa
# Operations:
#  - Init
#    - get container id
#    - create signal file
#    - get avail keyspaces
#  - Backup
#    - take snapshot
#    - upload snapshot data
#    - dump schema
#  - By trap
#    - remove signal file
#    - clear snapshot

set -euo pipefail

if [[ $# -ne 3 ]]; then
  echo "Usage: $0 <Node> <Target folder> <Path to ssh key>"
  exit 1
fi

source ./utils.sh

readonly node=$1
readonly targetFolder=$2
readonly containerName="scylla"
readonly nodeDataDir="/var/lib/scylla"
readonly sshKey=$3
readonly signalFilePath="$HOME/ctool/.voedgerbackup"

getContainer() {
    if ! containerID=$(utils_ssh -i "$sshKey" "$LOGNAME@$node" docker ps -q -f "name=$containerName"); then
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

signalFile() {
    case "$1" in
        create)
            # Create the directory if it doesn't exist
            utils_ssh -i "$sshKey" "$LOGNAME@$node" mkdir -p "$(dirname "$signalFilePath")"

            # Create the signal file
            utils_ssh -i "$sshKey" "$LOGNAME@$node" touch "$signalFilePath"
            echo "Signal file created at: $signalFilePath"
            ;;
        remove)
            # Remove the signal file if it exists
            if [ -e "$signalFilePath" ]; then
                utils_ssh -i "$sshKey" "$LOGNAME@$node" rm "$signalFilePath"
                echo "Signal file removed from: $signalFilePath"
            else
                echo "Signal file not found at: $signalFilePath"
            fi
            ;;
        *)
            echo "Invalid operation. Use 'create' or 'remove'."
            ;;
    esac
}

snapshotTag() {
    local timeFormat="%Y-%m-%d_%H-%M-%S"
    date +"$timeFormat"
}

snapshotCtl() {
    local operation="$1"

    local output
    local err

    if [ "$operation" != "snapshot" ] && [ "$operation" != "clearsnapshot" ]; then
      echo "Invalid operation. Use 'snapshot' or 'clearsnapshot'."
      return 1
    fi
    utils_ssh -i "$sshKey" "$LOGNAME@$node" "docker exec $container nodetool $operation -t $snapshotTag"
    echo "Snapshot $operation for node $node"
    err=$?

    if [ $err -ne 0 ]; then
        echo "Error taking snapshot: $output"
        return 1
    fi

    echo "$snapshotTag"
}

descKeyspaces() {
    containerID="$1"
    mapfile -t CQLout < <(
        utils_ssh -i "$sshKey" "$LOGNAME@$node" \
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

upload() {
    while [ $# -gt 0 ]; do
        local keyspace="$1"
            shift
        utils_ssh -i "$sshKey" "$LOGNAME@$node" "mkdir -p $targetFolder/$keyspace"
        cmd="cd $nodeDataDir/data && find . -type d -print0 | grep -z -iE '/$keyspace/[^/]+/snapshots/$snapshotTag' | tar -cvzf $targetFolder/$keyspace/data.tar.gz --null -T -"
            echo "Executing: $cmd"
        if ! utils_ssh -i "$sshKey" "$LOGNAME@$node" "$cmd"; then
            echo "Failed to upload data for keyspace $keyspace"
            return 1
        fi
    done
}

dump_schema() {
    while [ $# -gt 0 ]; do
        local keyspace="$1"
            shift
        cmd="docker exec $container cqlsh -e 'DESC KEYSPACE $keyspace' | grep -v '^$' | sed '/^Warning:/d' > $targetFolder/$keyspace/schema.cql"
        echo "Dump schema. Executing: $cmd"
        if ! utils_ssh -i "$sshKey" "$LOGNAME@$node" "$cmd"; then
            echo "Failed to dump schema for keyspace $keyspace"
            return 1
        fi
    done
}

backup() {
    signalFile create "$node"
    if container=$(getContainer "$containerName"); then
        echo "Container id for $containerName: $container"
    else
        echo "Failed to get Container id for $containerName"
    fi

    read -ra keyspaces <<< "$(descKeyspaces "$container")"

    echo "Init backup for node $node"
    for keyspace in "${keyspaces[@]}"; do
        echo "Keyspace: $keyspace"
    done

    snapshotTag=$(snapshotTag)
    if ! snapshotCtl snapshot; then
        echo "Failed to take snapshot"
        return 1
    fi

    upload "${keyspaces[@]}"
    dump_schema "${keyspaces[@]}"
}

function on_exit() {
    if [ -n "${snapshotTag:-}" ]; then
        snapshotCtl clearsnapshot
    fi
    signalFile remove "$node"
}

function on_error() {
    echo "Error occurred"
    on_exit
}

trap on_exit EXIT
trap on_error ERR

backup

exit 0



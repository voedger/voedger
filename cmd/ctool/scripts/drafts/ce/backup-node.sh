#!/usr/bin/env bash
# Copyright (c) 2023 Sigma-Soft, Ltd.
# @author Aleksei Ponomarev
#
# Backup scylla node.
# Usage: ./backupnode.sh <Target folder>
# Example: ./backupnode.sh /tmp/backup
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

set -Eeuo pipefail

if [[ $# -ne 1 ]]; then
  echo "Usage: $0 <Target folder>"
  exit 1
fi

source ./utils-cql.sh

readonly targetFolder=$1
readonly containerName="scylla"
readonly nodeDataDir="/var/lib/scylla"


signalFile() {
    case "$1" in
        create)
            # Create the directory if it doesn't exist
            mkdir -p "$(dirname "$signalFilePath")"

            # Create the signal file
            touch "$signalFilePath"
            echo "Signal file created at: $signalFilePath"
            ;;
        remove)
            # Remove the signal file if it exists
            if "[ -e $signalFilePath ]"; then
                rm "$signalFilePath"
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
    docker exec $container nodetool $operation -t $snapshotTag
    echo "Snapshot $operation for CENode"
    err=$?

    if [ $err -ne 0 ]; then
        echo "Error taking snapshot: $output"
        return 1
    fi

    echo "$snapshotTag"
}


upload() {
    while [ $# -gt 0 ]; do
        local keyspace="$1"
            shift
        mkdir -p $targetFolder/$keyspace
        cmd="cd $nodeDataDir/data && find . -type d -print0 | grep -z -iE '/$keyspace/[^/]+/snapshots/$snapshotTag' | tar -cvzf $targetFolder/$keyspace/data.tar.gz --null -T -"
            echo "Executing: $cmd"
        if ! eval "$cmd"; then
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
        if ! eval "$cmd"; then
            echo "Failed to dump schema for keyspace $keyspace"
            return 1
        fi
    done
}

backup() {
    signalFile create
    if container=$(getContainer "$containerName"); then
        echo "Container id for $containerName: $container"
    else
        echo "Failed to get Container id for $containerName"
    fi

    read -ra keyspaces <<< "$(descKeyspaces "$container")"

    echo "Init backup for CENode"
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
    signalFile remove
}

function on_error() {
    echo "Error occurred"
    on_exit
}

trap on_exit EXIT
trap on_error ERR

backup

exit 0



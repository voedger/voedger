#!/usr/bin/env bash
#
# Copyright (c) 2024 Sigma-Soft, Ltd.
# @author Dmitry Molchanovsky
#
# copying a local file to remote hosts

set -Eeuo pipefail
set -x

if [ $# -lt 3 ]; then
    echo "Usage: $0 <lical file> <remote file> <remote host1> [<remote host2> ...]"
    exit 1
fi

source ./utils.sh

SSH_USER=$LOGNAME
LOCAL_FILE=$1
REMOTE_FILE=$2
shift 2

while [ $# -gt 0 ]; do
    REMOTE_HOST=$1

    utils_scp ${LOCAL_FILE} ${SSH_USER}@${REMOTE_HOST}:${REMOTE_FILE}

    if [ $? -eq 0 ]; then
        echo "The file ${LOCAL_FILE} is successfully copied from a local machine to a remote host ${REMOTE_HOST}."
    else
        echo "An error when copying a file ${LOCAL_FILE} to a remote host ${REMOTE_HOST}."
        exit 1
    fi

    shift
done

set +x

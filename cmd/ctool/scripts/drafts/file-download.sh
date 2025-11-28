#!/usr/bin/env bash
#
# Copyright (c) 2024 Sigma-Soft, Ltd.
# @author Dmitry Molchanovsky
#
# copying a file from a remote host to a local

set -Eeuo pipefail
set -x


if [ $# -ne 3 ]; then
    echo "Usage: $0 <remote host> <remote file> <local file>"
    exit 1
fi

source ./utils.sh

SSH_USER=$LOGNAME
REMOTE_HOST=$1
REMOTE_FILE=$2
LOCAL_FILE=$3

utils_scp ${SSH_USER}@${REMOTE_HOST}:${REMOTE_FILE} ${LOCAL_FILE}

if [ $? -eq 0 ]; then
    echo "File ${REMOTE_FILE} is successfully copied from a remote host ${REMOTE_HOST} to a local machine."
else
    echo "An error when copying a file ${REMOTE_FILE} from a remote host ${REMOTE_HOST}. "
    exit 1
fi

set +x

#!/usr/bin/env bash
#
# Copyright (c) 2024 Sigma-Soft, Ltd.
# @author Dmitry Molchanovsky
#
# Restanation of the service in a remote host

set -Eeuo pipefail
set -x

if [ $# -ne 2 ]; then
    echo "Usage: $0 <remote host> <service name>"
    exit 1
fi

source ./utils.sh

REMOTE_HOST=$1
SSH_USER=$LOGNAME
SERVICE_NAME=$2

RESTART_COMMAND="docker service ls --format '{{.Name}}' | grep '${SERVICE_NAME}' | xargs -I {} docker service update --force {}"
echo "RESTART_COMMAND=$RESTART_COMMAND"

utils_ssh "${SSH_USER}@${REMOTE_HOST}" "${RESTART_COMMAND}"

set +x
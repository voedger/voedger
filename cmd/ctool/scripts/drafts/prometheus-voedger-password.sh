#!/usr/bin/env bash
#
# Copyright (c) 2024 Sigma-Soft, Ltd.
# @author Dmitry Molchanovsky
#
#
# Sets a new password for the admin user in Prometheus
# on specified remote hosts
set -Eeuo pipefail
set -x
if [ $# -lt 3 ]; then
    echo "Usage: $0 <password> <hashed password> <host1> [<host2> ...]"
    exit 1
fi

source ./utils.sh

NEW_PASSWORD=$1
HASHED_PASSWORD=$2
SSH_USER=$LOGNAME
USER_NAME="voedger"

shift 2
host_index=1
for host in "$@"; do
    APP_NODE_CONTAINER=$(utils_ssh ${SSH_USER}@${host} "docker ps --format '{{.Names}}' | grep prometheus")
    if [ -z "$APP_NODE_CONTAINER" ]; then
        echo "Prometheus container was not found on a host ${host}"
    else
        utils_ssh ${SSH_USER}@${host} "sed -i 's/${USER_NAME}:.*/${USER_NAME}: ${HASHED_PASSWORD//\//\\/}/' ~/prometheus/web.yml"
        echo "Password for voedger user in Prometheus on ${host} was successfully changed"

	if [ "$host_index" -eq 1 ]; then
          SERVICE_NAME="MonDockerStack_prometheus1"
        else
          SERVICE_NAME="MonDockerStack_prometheus2"
        fi

            utils_ssh ${SSH_USER}@${host} "docker service update '${SERVICE_NAME}' --force"

            while true; do
                if utils_ssh "${SSH_USER}@${host}" "docker service ps '${SERVICE_NAME}' | grep 'Running' >/dev/null"; then
                    echo "Service ${SERVICE_NAME} is up and running."
                    break
                else
                    echo "Service ${SERVICE_NAME} is not yet ready, waiting..."
                    sleep 5
                fi
            done

    	    echo "${SERVICE_NAME} service was updated"
   fi
   host_index=$((host_index + 1))
done
set +x
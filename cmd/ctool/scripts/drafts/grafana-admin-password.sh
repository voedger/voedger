#!/usr/bin/env bash
#
# Copyright (c) 2024 Sigma-Soft, Ltd.
# @author Dmitry Molchanovsky
#
#
# Sets a new password for the admin user in Grafana
# on a app-node-1 and app-node-2 hosts
set -Eeuo pipefail

if [ $# -ne 2 ]; then
    echo "Usage: $0 <password> <app-node host>"
    exit 1
fi

source ./utils.sh

NEW_PASSWORD=$1
APP_NODE_HOST=$2
SSH_USER=$LOGNAME

APP_NODE_CONTAINER=$(utils_ssh ${SSH_USER}@${APP_NODE_HOST} "docker ps --format '{{.Names}}' | grep grafana")

if [ -z "$APP_NODE_CONTAINER" ]; then
    echo "Grafana container was not found on a host ${APP_NODE_HOST}"
    exit 1
fi

utils_ssh ${SSH_USER}@${APP_NODE_HOST} "docker exec ${APP_NODE_CONTAINER} grafana-cli admin reset-admin-password ${NEW_PASSWORD}"
echo "Password for admin user in Grafana on ${APP_NODE_HOST} was successfully changed"


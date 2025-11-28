#!/usr/bin/env bash
# Copyright (c) 2024 Sigma-Soft, Ltd.
# @author Aleksei Ponomarev

set -Eeuo pipefail

set -x

domain=$1

IFS=' ' read -r -a ssh_options_args <<< "$SSH_OPTIONS"

PROMETHEUS_RESPONSE=$(curl -sL -w '%{http_code}' -u "$MON_USER:$MON_PASSWORD" -o /dev/null "$domain"/prometheus/-/healthy)
if [[ "${PROMETHEUS_RESPONSE}" == "200" ]]; then
    echo "Prometheus is up and running on node."
else
    echo "Failed to reach Prometheus on node. HTTP response code: ${PROMETHEUS_RESPONSE}"
    exit 1
fi

ALERTMANAGER_RESPONSE=$(ssh "${ssh_options_args[@]}" ubuntu@"$PUBLIC_IP" "\
    curl -sL -w '%{http_code}' -u $MON_USER:$MON_PASSWORD \
    -o /dev/null http://db-node-1:9093/-/healthy")
if [[ "${ALERTMANAGER_RESPONSE}" == "200" ]]; then
    echo "Alertmanager is up and running on node"
else
    echo "Failed to reach Alertmanager on node. HTTP response code: ${ALERTMANAGER_RESPONSE}"
    exit 1
fi

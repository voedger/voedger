#!/usr/bin/env bash
#
# Copyright (c) 2024 Sigma-Soft, Ltd.
# @author Dmitry Molchanovsky
#
# sending event to alert manager

set -Eeuo pipefail
set -x


if [ $# -ne 1 ]; then
    echo "Usage: $0 <event>"
    exit 1
fi


HDR1="Host: app-node-1:9093"
HDR2="User-Agent: Alertmanager/0.25.0"
HDR3="Content-Type: application/json"
EVENT=$1

curl -XPOST -v -H "${HDR1}" -H "${HDR2}" -H "${HDR3}" -d @${EVENT} http://app-node-1:9093/api/v2/alerts

echo  "the alert was sent successfully"

set +x
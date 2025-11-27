#!/usr/bin/env bash
# Copyright (c) 2024 Sigma-Soft, Ltd.
# @author Aleksei Ponomarev

set -Eeuo pipefail

set -x

MON_PASSWORD=$1

ctool_cmd="cd /home/ubuntu/voedger/cmd/ctool && ./ctool mon password -v "

if [ "$CLUSTER_TYPE" == "n5" -o "$CLUSTER_TYPE" == "se3" ]; then
    ctool_cmd+="--ssh-key /tmp/amazonKey.pem "
fi
ctool_cmd+="$MON_PASSWORD"

IFS=' ' read -r -a ssh_options_args <<< "$SSH_OPTIONS"

ssh "${ssh_options_args[@]}" ubuntu@"$CTOOL_IP" "$ctool_cmd"

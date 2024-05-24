#!/usr/bin/env bash
# Copyright (c) 2024 Sigma-Soft, Ltd.
# @author Aleksei Ponomarev

set -euo pipefail

set -x

MON_PASSWORD=$1

IFS=' ' read -r -a ssh_options_args <<< "$SSH_OPTIONS"

ssh "${ssh_options_args[@]}" ubuntu@"$CTOOL_IP" "cd /home/ubuntu/voedger/cmd/ctool && ./ctool mon password -v --ssh-key /tmp/amazonKey.pem $MON_PASSWORD"

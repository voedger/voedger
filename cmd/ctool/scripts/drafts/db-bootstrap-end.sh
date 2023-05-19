#!/usr/bin/env bash
#
# Copyright (c) 2023 Sigma-Soft, Ltd.
# @author Aleksei Ponomarev
# 
# Delete line that contain 'replace_address_first_boot: <ip address>'
# after bootstrap procedure

set -euo pipefail

set -x

if [[ $# -ne 1 ]]; then
  echo "Usage: $0 <new node>"
  exit 1
fi


SSH_USER=$LOGNAME
SSH_OPTIONS='-o UserKnownHostsFile=/dev/null -o StrictHostKeyChecking=no -o LogLevel=ERROR'

pattern="replace_address_first_boot"

ssh $SSH_OPTIONS $SSH_USER@$1 sed -i "/$pattern/d" ~/scylla/scylla.yaml

set +x

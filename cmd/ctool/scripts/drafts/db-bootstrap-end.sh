#!/usr/bin/env bash
#
# Copyright (c) 2023 Sigma-Soft, Ltd.
# @author Aleksei Ponomarev
#
# Delete line that contain 'replace_address_first_boot: <ip address>'
# after bootstrap procedure

set -Eeuo pipefail

set -x

if [[ $# -ne 1 ]]; then
  echo "Usage: $0 <new node>"
  exit 1
fi

source ./utils.sh

SSH_USER=$LOGNAME

pattern="replace_address_first_boot"

utils_ssh "$SSH_USER@$1" sed -i "/$pattern/d" ~/scylla/scylla.yaml

set +x

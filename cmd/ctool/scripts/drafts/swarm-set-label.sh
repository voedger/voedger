#!/usr/bin/env bash
#
# Copyright (c) 2023 Sigma-Soft, Ltd.
# @author Aleksei Ponomarev
#
# Set labels for host

set -euo pipefail

set -x

if [[ $# -ne 4 ]]; then
  echo "Usage: $0 <swarm manager ip address> <node ip address> <key for label> <value for label>"
  exit 1
fi

source ./utils.sh

SSH_USER=$LOGNAME

REMOTE_HOSTNAME=$(utils_ssh "$SSH_USER@$2" 'hostname')

# Set node label
utils_ssh "$SSH_USER@$1" "docker node update --label-add $3=$4 $REMOTE_HOSTNAME"

set +x

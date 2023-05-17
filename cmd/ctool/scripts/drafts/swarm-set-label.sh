#!/usr/bin/env bash
#
# Copyright (c) 2023 Sigma-Soft, Ltd.
# @author Aleksei Ponomarev
#
# Set labels for host

set -euo pipefail

set -x

if [[ $# -ne 3 ]]; then
  echo "Usage: $0 <swarm manager ip address> <node ip address> <type for label>"
  exit 1
fi

SSH_USER=$LOGNAME
SSH_OPTIONS='-o UserKnownHostsFile=/dev/null -o StrictHostKeyChecking=no -o LogLevel=ERROR'

REMOTE_HOSTNAME=$(ssh $SSH_OPTIONS $SSH_USER@$2 'hostname')

# Set node label
ssh $SSH_OPTIONS $SSH_USER@$1 "docker node update --label-add type=$3 $REMOTE_HOSTNAME"

set +x

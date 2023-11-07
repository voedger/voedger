#!/usr/bin/env bash
#
# Copyright (c) 2023 Sigma-Soft, Ltd.
# @author Aleksei Ponomarev
#
# Add node to Swarm. Find node id with dedicated ip. If node id not found - join node to swarm cluster.
# Token, stored in 'manager.token' file used for join node.

set -euo pipefail

set -x

if [ "$#" -lt 2 ]; then
  echo "Usage: $0 <node ip address> <node name>" >&2
  exit 1
fi

source ./utils.sh

SSH_USER=$LOGNAME

utils_ssh $SSH_USER@$1 sudo hostnamectl set-hostname $2

set +x

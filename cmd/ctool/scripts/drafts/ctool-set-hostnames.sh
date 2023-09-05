#!/usr/bin/env bash
#
# Copyright (c) 2023 Sigma-Soft, Ltd.
# @author Aleksei Ponomarev
#
# Add node to Swarm. Find node id with dedicated ip. If node id not found - join node to swarm cluster.
# Token, stored in 'manager.token' file used for join node.

set -euo pipefail

set +x

if [ "$#" -lt 5 ]; then
  echo "Usage: $0 <AppNode1> <AppNode2> <DBNode1> <DBNode2> <DBNode3>" >&2
  exit 1
fi

hosts=("AppNode1" "AppNode2" "DBNode1" "DBNode2" "DBNode3")

SSH_USER=$LOGNAME
SSH_OPTIONS='-o UserKnownHostsFile=/dev/null -o StrictHostKeyChecking=no -o LogLevel=ERROR'

i=0
while [ $# -gt 0 ]; do

  ssh $SSH_OPTIONS $SSH_USER@$1 sudo hostnamectl set-hostname ${hosts[i]}
  shift
  ((++i))

done

set +x

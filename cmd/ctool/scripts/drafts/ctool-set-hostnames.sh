#!/usr/bin/env bash
#
# Copyright (c) 2023 Sigma-Soft, Ltd.
# @author Aleksei Ponomarev
# set hostnames for all node in voedger cluster
set -euo pipefail

set +x

if [ "$#" -lt 5 ]; then
  echo "Usage: $0 <app-node-1> <app-node-2> <db-node-1> <db-node-2> <db-node-3>" >&2
  exit 1
fi

hosts=("app-node-1" "app-node-2" "db-node-1" "db-node-2" "db-node-3")

SSH_USER=$LOGNAME
SSH_OPTIONS='-o UserKnownHostsFile=/dev/null -o StrictHostKeyChecking=no -o LogLevel=ERROR'

i=0
while [ $# -gt 0 ]; do

  ssh $SSH_OPTIONS $SSH_USER@$1 sudo hostnamectl set-hostname ${hosts[i]}
  shift
  ((++i))

done

set +x

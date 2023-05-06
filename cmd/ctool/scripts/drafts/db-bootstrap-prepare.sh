#!/usr/bin/env bash
#
# Copyright (c) 2023 Sigma-Soft, Ltd.
# @author Aleksei Ponomarev
# 
# Prepare scylla node for bootstrap procedure: add 'replace_address_first_boot: <ip address>'
# this tell scylla cluster that this new node will set up instead node in command above.
# And bootstrap wiil start immediately after scylla load.

set -euo pipefail

set -x

if [[ $# -ne 2 ]]; then
  echo "Usage: $0 <lost node> <new node>"
  exit 1
fi


SSH_USER=$LOGNAME
SSH_OPTIONS='-o UserKnownHostsFile=/dev/null -o StrictHostKeyChecking=no'

echo "replace_address_first_boot: $1" | ssh $SSH_OPTIONS $SSH_USER@$2 'cat >> ~/scylla/scylla.yaml'

set +x

#!/usr/bin/env bash
#
# Copyright (c) 2023 Sigma-Soft, Ltd.
# @author Aleksei Ponomarev

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

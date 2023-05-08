#!/usr/bin/env bash
#
# Copyright (c) 2023 Sigma-Soft, Ltd.
# @author Aleksei Ponomarev

set -euo pipefail

set -x

if [[ $# -ne 1 ]]; then
  echo "Usage: $0 <node ip address>"
  exit 1
fi


SSH_USER=$LOGNAME
SSH_OPTIONS='-o UserKnownHostsFile=/dev/null -o StrictHostKeyChecking=no'

ssh $SSH_OPTIONS $SSH_USER@$1 "sudo mkdir -p /var/lib/scylla && mkdir -p ~/scylla"
cat ./scylla.yaml | ssh $SSH_OPTIONS $SSH_USER@$1 'cat > ~/scylla/scylla.yaml'

set +x

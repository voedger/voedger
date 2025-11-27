#!/usr/bin/env bash
#
# Copyright (c) 2023 Sigma-Soft, Ltd.
# @author Aleksei Ponomarev
#
# Prepare scylla node for bootstrap procedure: add 'replace_address_first_boot: <ip address>'
# this tell scylla cluster that this new node will set up instead node in command above.
# And bootstrap wiil start immediately after scylla load.

set -Eeuo pipefail

set -x

if [[ $# -ne 2 ]]; then
  echo "Usage: $0 <lost node> <new node>"
  exit 1
fi

source ./utils.sh

SSH_USER=$LOGNAME

echo "replace_address_first_boot: $1" | utils_ssh "$SSH_USER@$2" 'cat >> ~/scylla/scylla.yaml'

set +x

#!/usr/bin/env bash
#
# Copyright (c) 2023 Sigma-Soft, Ltd.
# @author Aleksei Ponomarev
#
# Set labels for host

set -Eeuo pipefail

set -x

if [[ $# -ne 4 ]]; then
  echo "Usage: $0 <swarm manager ip address> <node ip address> <key for label> <value for label>"
  exit 1
fi

source ./utils.sh

SSH_USER=$LOGNAME

# $1 is the swarm manager IP address (for SSH connection to run docker commands)
# $2 is the node IP address (to get the hostname of the node being labeled)
# $3 is the label key
# $4 is the label value

# Get the hostname of the node we're labeling
REMOTE_HOSTNAME=$(utils_ssh "$SSH_USER@$2" 'hostname')

# Set node label - SSH to the swarm manager to run the docker node update command
utils_ssh "$SSH_USER@$1" "docker node update --label-add $3=$4 $REMOTE_HOSTNAME"

set +x

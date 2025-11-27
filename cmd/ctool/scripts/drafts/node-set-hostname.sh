#!/usr/bin/env bash
#
# Copyright (c) 2024 Sigma-Soft, Ltd.
# @author Aleksei Ponomarev
#
# Set hostname with hostnamectl

set -Eeuo pipefail

set -x

if [ "$#" -lt 2 ]; then
  echo "Usage: $0 <node ip address> <node name>" >&2
  exit 1
fi

source ./utils.sh

SSH_USER=$LOGNAME

utils_ssh $SSH_USER@$1 sudo hostnamectl set-hostname $2

set +x

#!/usr/bin/env bash
#
# Copyright (c) 2024 Sigma-Soft, Ltd.
# @author Dmitry Molchanovsky
#
# checks that the host is down in the Swarm cluster

set -Eeuo pipefail

if [ $# -ne 2 ]; then
  echo "Usage: $0 <IP-Address> <docker host name>"
  exit 1
fi

HOST=$1
HOST_NAME=$2

if ./host-check.sh ${HOST} ${HOST_NAME}; then
  echo "host ${HOST} is live"
  exit 1
else
  echo "host ${HOST} is down"
  exit 0
fi


#!/usr/bin/env bash
#
# Copyright (c) 2024 Sigma-Soft, Ltd.
# @author Dmitry Molchanovsky
# 
# checks that the host is down in the Swarm cluster

set -euo pipefail

if [ $# -ne 1 ]; then
  echo "Usage: $0 <IP-Address>" 
  exit 1
fi

HOST=$1

if ./host-check.sh ${HOST}; then
  echo "host ${HOST} is live"
  exit 1
else
  echo "host ${HOST} is down"
  exit 0
fi


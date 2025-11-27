#!/usr/bin/env bash
#
# Copyright (c) 2023 Sigma-Soft, Ltd.
# @author Dmitry Molchanovsky
#
# pinging the address of the host
# checks that the host is alive in the Swarm cluster

set -Eeuo pipefail


if [ $# -lt 1 ] || [ $# -gt 2 ]; then
  echo "Usage: $0 <IP-Address> ["only-ping" or docker host name]"
  exit 1
fi

# Verification of the availability of the specified IP address using Ping
if ping -c 1 "$1" >/dev/null; then
  echo "IP address $1 is available."
else
  echo "IP address $1 is not available."
  exit 1
fi

if [ $# -eq 2 ] && [ "$2" == "only-ping" ]; then
  exit 0
fi


# Checking the availability of docker on the host
if ! command -v docker &> /dev/null; then
  echo "Docker is not installed"
  exit 1
fi

# Checking the state of the node in the Swarm Claster
node_status=$(docker node ls --format '{{json .}}' | jq -r "select(.Hostname == \"$2\") | .Status")

if [ -z "$node_status" ]; then
  echo "The indicated node was not found in the Swarm Claster."
  exit 1
fi

if [ "$node_status" = "Ready" ]; then
  echo "Node $2 is in working condition."
else
  echo "Node $2 is not in working condition."
  exit 1
fi

set +x

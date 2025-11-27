#!/usr/bin/env bash
#
# Copyright (c) 2023 Sigma-Soft, Ltd.
# @author Aleksei Ponomarev
#
# Deploy scylla cluster
#   - Install docker
#   - Init swarm cluster
#   - join dedicated nodes to swarm
#   - add labels to swarm nodes
#   - prepare hosts for scylla services
#   - deploy scylla stack

set -Eeuo pipefail

set -x

if [[ $# -ne 3 ]]; then
  echo "Usage: $0 <db-node-1> <db-node-2> <db-node-3>"
  exit 1
fi


# Save a copy of the original arguments
args=("$@")

# Install Docker on hosts
while [ $# -gt 0 ]; do
  source ./docker-install.sh $1
  shift
done

# Restore the original arguments
set -- "${args[@]}"

# Init swarm mode
./swarm-init.sh $1
./swarm-set-label.sh $1 $1 "db-node-1" "true"
./db-node-prepare.sh $1
MANAGER=$1

shift

# Add remaining nodes to swarm cluster
i=2
while [ $# -gt 0 ]; do
  ./swarm-add-node.sh $MANAGER $1
  ./swarm-set-label.sh $MANAGER $1 "db-node-$i" "true"
  ./db-node-prepare.sh $1
  shift
  ((i++))
done

# Restore the original arguments
set -- "${args[@]}"

# Start db cluster
./db-cluster-start.sh $@


set +x

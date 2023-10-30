#!/usr/bin/env bash
#
# Copyright (c) 2023 Sigma-Soft, Ltd.
# @author Aleksei Ponomarev
# 
# Prepare scylla node: create directory for scylla data files,
# and copy scylla config to host

set -euo pipefail

set -x

if [[ $# -ne 2 ]]; then
  echo "Usage: $0 <db-node> <datacenter>"
  exit 1
fi


SSH_USER=$LOGNAME
SSH_OPTIONS='-o UserKnownHostsFile=/dev/null -o StrictHostKeyChecking=no -o LogLevel=ERROR'

dc=$2
rackdc="
#
# cassandra-rackdc.properties
#
# The lines may include white spaces at the beginning and the end.
# The rack and data center names may also include white spaces.
# All trailing and leading white spaces will be trimmed.
#
dc=$dc
rack=rack1
prefer_local=true
# dc_suffix=<Data Center name suffix, used by EC2SnitchXXX snitches>
#
"

ssh $SSH_OPTIONS $SSH_USER@$1 "sudo mkdir -p /var/lib/scylla && mkdir -p ~/scylla"
cat ./scylla.yaml | ssh $SSH_OPTIONS $SSH_USER@$1 'cat > ~/scylla/scylla.yaml'
echo "$rackdc" | ssh $SSH_OPTIONS $SSH_USER@$1 'cat > ~/scylla/cassandra-rackdc.properties'

set +x

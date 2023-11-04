#!/usr/bin/env bash
#
# Copyright (c) 2023 Sigma-Soft, Ltd.
# @author Aleksei Ponomarev
# 
# Prepare scylla node: create directory for scylla data files,
# and copy scylla config to host

set -euo pipefail

set -x

if [[ $# -lt 1 ]]; then
  echo "Usage: $0 <db-node> <datacenter>"
  exit 1
fi

source ./utils.sh

SSH_USER=$LOGNAME
SSH_OPTIONS='-o UserKnownHostsFile=/dev/null -o StrictHostKeyChecking=no -o LogLevel=ERROR'

function addVolumeDC() {
    local VOL_DC="$HOME/scylla/cassandra-rackdc.properties:/etc/scylla/cassandra-rackdc.properties"
    local SERVICES=$(yq eval '.services | keys | map(select(test("^scylla"))) | .[]' docker-compose.yml)

    for SERVICE in $SERVICES; do
        local VOLUME_EXISTS=$(yq eval ".services.$SERVICE.volumes | .[] | select(. == \"$VOL_DC\")" docker-compose.yml)

        if [ -z "$VOLUME_EXISTS" ]; then
#            yq --inplace docker-compose-template.yml "services.$SERVICE.volumes[+]" "$VOL_DC"
            yq eval --inplace ".services.$SERVICE.volumes += [\"$VOL_DC\"]" docker-compose.yml
            echo "Add to service '$SERVICE': $VOL_DC"
        else
            echo "Already present in service '$SERVICE': $VOL_DC"
        fi
    done
}


utils_ssh "$SSH_USER@$1" "sudo mkdir -p /var/lib/scylla && mkdir -p ~/scylla && mkdir -p ~/scylla.d"

if [ -n "${2+x}" ] && [ -n "$2" ]; then
dc=$2
rackdc="#
# cassandra-rackdc.properties
#
# The lines may include white spaces at the beginning and the end.
# The rack and data center names may also include white spaces.
# All trailing and leading white spaces will be trimmed.
#
dc=$dc
rack=rack1
# prefer_local=true
# dc_suffix=<Data Center name suffix, used by EC2SnitchXXX snitches>
#
"
addVolumeDC
sed -i 's/endpoint_snitch: SimpleSnitch/endpoint_snitch: GossipingPropertyFileSnitch/' ./scylla.yaml
echo "$rackdc" | utils_ssh "$SSH_USER@$1" 'cat > ~/scylla/cassandra-rackdc.properties'
fi

io_conf="SEASTAR_IO=\"--io-properties-file=/etc/scylla.d/io_properties.yaml\""
# DO NO EDIT
# This file should be automatically configure by scylla_io_setup
#
# SEASTAR_IO=\"--max-io-requests=1 --num-io-queues=1\"

io_properties="disks:
  - mountpoint: /var/lib/scylla
    read_iops: 680915
    read_bandwidth: 3577784832
    write_iops: 94199
    write_bandwidth: 609521344"

cpuset_conf="# DO NO EDIT
# This file should be automatically configure by scylla_cpuset_setup
#
# CPUSET=\"--cpuset 0 --smp 1\""

memory_conf="# DO NO EDIT
# This file should be automatically configure by scylla_memory_setup
#
# MEM_CONF=--lock-memory=1"

dev_mode_conf=""

cat ./scylla.yaml | utils_ssh "$SSH_USER@$1" 'cat > ~/scylla/scylla.yaml'

echo "$io_properties" | utils_ssh "$SSH_USER@$1" 'test -e ~/scylla.d/io_properties.yaml || cat > ~/scylla.d/io_properties.yaml'
echo "$io_conf" | utils_ssh "$SSH_USER@$1" 'test -e ~/scylla.d/io.conf || cat > ~/scylla.d/io.conf'
echo "$cpuset_conf" | utils_ssh "$SSH_USER@$1" 'test -e ~/scylla.d/cpuset.conf || cat > ~/scylla.d/cpuset.conf'
echo "$memory_conf" | utils_ssh "$SSH_USER@$1" 'test -e ~/scylla.d/memory.conf || cat > ~/scylla.d/memory.conf'
echo "$dev_mode_conf" | utils_ssh "$SSH_USER@$1" 'test -e ~/scylla.d/dev-mode.conf || cat > ~/scylla.d/dev-mode.conf'

set +x

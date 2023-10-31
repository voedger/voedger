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


SSH_USER=$LOGNAME
SSH_OPTIONS='-o UserKnownHostsFile=/dev/null -o StrictHostKeyChecking=no -o LogLevel=ERROR'

function addVolumeDC() {
    local VOL_DC="$HOME/scylla/cassandra-rackdc.properties:/etc/scylla/cassandra-rackdc.properties"
    local SERVICES=$(yq eval '.services | keys | map(select(test("^scylla"))) | .[]' docker-compose-template.yml)

    for SERVICE in $SERVICES; do
        local VOLUME_EXISTS=$(yq eval ".services.$SERVICE.volumes | .[] | select(. == \"$VOL_DC\")" docker-compose-template.yml)

        if [ -z "$VOLUME_EXISTS" ]; then
            yq w -i docker-compose-template.yml "services.$SERVICE.volumes[+]" "$VOL_DC"
            echo "Add to service '$SERVICE': $VOL_DC"
        else
            echo "Already present in service '$SERVICE': $VOL_DC"
        fi
    done
}


ssh $SSH_OPTIONS $SSH_USER@$1 "sudo mkdir -p /var/lib/scylla && mkdir -p ~/scylla"

if [ -n "${2+x}" ] && [ -n "$2" ]; then
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
addVolumeDC
sed -i 's/endpoint_snitch: SimpleSnitch/endpoint_snitch: GossipingPropertyFileSnitch/' ./scylla.yaml
echo "$rackdc" | ssh $SSH_OPTIONS $SSH_USER@$1 'cat > ~/scylla/cassandra-rackdc.properties'
fi

cat ./scylla.yaml | ssh $SSH_OPTIONS $SSH_USER@$1 'cat > ~/scylla/scylla.yaml'

set +x

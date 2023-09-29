#!/usr/bin/env bash
# Usage: ./update-service.sh <lost_node> <new_node>
# 
# Update scylla ctack before replace node(edit only the service whose node was lost):
#   - remove lost node ip address from seed list
#   - replace broadcast-address parameter with ip address node instead
#   - replace broadcast-rpc-address parameter with ip address node instead
#   - update healthcheck with new ip address node instead
# Service find by ip adddres bind to halthcheck

set -euo pipefail

if [ $# -ne 1 ]; then
#  echo "Usage: $0 <lost_node> <new_node>"
   echo "Usage: $0 <cluster node name>"
  exit 1
fi

#lost_node=$1
#new_node=$2

function modify_clp {
 # $1 - cluster node name
 # $2 - command

 # $1 - lost node
 # $2 - new_node
 # $3 - input command looks like "command: --seeds 10.0.0.12,10.0.0.13,10.0.0.14 --listen-address 0.0.0.0 --broadcast-address 10.0.0.12 --broadcast-rpc-address 10.0.0.12"

  # Remove the lost_node from the seeds list
  seeds=$(echo $2 | sed "s/\(--seeds [^ ,]*,\)\?\(,\)\?$1\(,\)\?/\1/")

  # Replace the IP address in broadcast-address and broadcast-rpc-address with new_node
  # output_cmd=$(echo "$seeds" | sed -E "s/--broadcast-address[[:space:]]+$1/--broadcast-address $2/; s/--broadcast-rpc-address[[:space:]]+$1/--broadcast-rpc-address $2/;")

  # echo "$output_cmd"
  echo "$seeds"
}


# Find the service name that matches the lost_node IP address
service_name=$(yq e '.services | to_entries | .[] | select(.value.healthcheck.test[] | contains("'$1'")) | .key' docker-compose.yml)

if [ -z "$service_name" ]; then
  echo "Could not find a service with IP address $1"
  exit 1
fi

echo "Updating service $service_name"

scy_cmd=$(yq eval '.services.'$service_name'.command' docker-compose.yml)
echo "Updating scylla command '$scy_cmd'"

# new_scy_cmd=$(modify_clp $lost_node $new_node "$scy_cmd")

new_scy_cmd=$(modify_clp $1 "$scy_cmd")
echo "With new command '$new_scy_cmd'"

export new_scy_cmd; yq eval --inplace --prettyPrint '.services."'$service_name'".command = strenv(new_scy_cmd)' docker-compose.yml

#echo "Update healthcheck ip address with new one"
#yq eval --inplace '.services."'$service_name'".healthcheck = {"test": ["CMD-SHELL", "nodetool status | awk '\''/^UN/ {print $$2}'\'' | grep -w '\'''$new_node''\''"], "interval": "15s", "timeout": "5s", "retries": 90}' docker-compose.yml

echo "Service '$service_name' updated successfully"
echo "$service_name"

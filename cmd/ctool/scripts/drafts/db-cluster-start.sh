#!/usr/bin/env bash
#
# Copyright (c) 2023 Sigma-Soft, Ltd.
# @author Aleksei Ponomarev
#
# Create docker-compose.yml for scylla stack and deploy

set -euo pipefail

set -x

if [[ $# -ne 3 ]]; then
  echo "Usage: $0 <DBNode1> <DBNode2> <DBNode3>"
  exit 1
fi

SSH_USER=$LOGNAME
SSH_OPTIONS='-o UserKnownHostsFile=/dev/null -o StrictHostKeyChecking=no -o LogLevel=ERROR'

hosts=("DBNode1" "DBNode2" "DBNode3")

# Function to update /etc/hosts on a remote host using SSH
update_hosts_file() {
  local host="$1"
  local ip="$2"

  # SSH command to execute on the remote host
  ssh $SSH_OPTIONS $SSH_USER@$ip "sudo bash -c 'echo -e \"$ip\t$host\" >> /etc/hosts'"
}

# Prepare for name resolving - iterate over each hostname and update /etc/hosts on each host
for host in "${hosts[@]}"; do
  # Inner loop: Iterate over the three IP addresses
  for ip_address in "$@"; do
    update_hosts_file "$host" "$ip_address"
  done
done

DBNode1=$1
DBNode2=$2
DBNode3=$3

# Replace the template values in the YAML file with the arguments (scylla nodes ip addresses)
# and store as prod compose file for start swarm services
cat docker-compose-template.yml | \
    sed "s/{{\.${hosts[0]}}}/${hosts[0]}/g; s/{{\.${hosts[1]}}}/${hosts[1]}/g; s/{{\.${hosts[2]}}}/${hosts[2]}/g" \
    > ./docker-compose.yml

cat ./docker-compose.yml | ssh $SSH_OPTIONS $SSH_USER@$1 'cat > ~/docker-compose.yml'

ssh $SSH_OPTIONS $SSH_USER@$1 "docker stack deploy --compose-file ~/docker-compose.yml DBDockerStack"

set +x

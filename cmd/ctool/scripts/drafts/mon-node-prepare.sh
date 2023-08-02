#!/usr/bin/env bash
#
# Copyright (c) 2023 Sigma-Soft, Ltd.
# @author Aleksei Ponomarev
# 
# Prepare scylla node: create directory for scylla data files,
# and copy scylla config to host

set -euo pipefail

set -x

if [[ $# -ne 5 ]]; then
  echo "Usage: $0 <AppNode1> <AppNode2> <DBNode1> <DBNode2> <DBNode3>" 
  exit 1
fi

SSH_USER=$LOGNAME
SSH_OPTIONS='-o UserKnownHostsFile=/dev/null -o StrictHostKeyChecking=no -o LogLevel=ERROR'

AppNode1=$1
AppNode2=$2
DBNode1=$3
DBNode2=$4
DBNode3=$5

hosts=("AppNode1" "AppNode2" "DBNode1" "DBNode2" "DBNode3")

update_hosts_file() {
  local host=$1
  local ip=$2
  local hr=$3 
  # SSH command to execute on the remote host
  ssh $SSH_OPTIONS $SSH_USER@$ip "sudo bash -c 'echo -e \"$hr\t$host\" >> /etc/hosts'"
}

args_array=("$@")
# Prepare for name resolving - iterate over each hostname and update /etc/hosts on each host
i=0
for host in "${hosts[@]}"; do
  ip=${args_array[i]}
  # Iterate over the three IP addresses
  for ip_address in "$@"; do
      update_hosts_file $host $ip_address $ip
  done
((++i))
done

count=0

while [ $# -gt 0 ] && [ $count -lt 2 ]; do
  echo "Processing: $1"
  ssh $SSH_OPTIONS $SSH_USER@$1 "bash -s" << EOF
  sudo mkdir -p /prometheus && mkdir -p ~/prometheus;
  sudo mkdir -p /alertmanager && mkdir -p ~/alertmanager;
  mkdir -p ~/grafana/provisioning/dashboards;
  mkdir -p ~/grafana/provisioning/datasources;
  sudo mkdir -p /var/lib/grafana;
EOF

   cat ./grafana/provisioning/dashboards/swarmprom-nodes-dash.json | \
      ssh $SSH_OPTIONS $SSH_USER@$1 'cat > ~/grafana/provisioning/dashboards/swarmprom-nodes-dash.json'
   cat ./grafana/provisioning/dashboards/swarmprom-prometheus-dash.json | \
      ssh $SSH_OPTIONS $SSH_USER@$1 'cat > ~/grafana/provisioning/dashboards/swarmprom-prometheus-dash.json'
   cat ./grafana/provisioning/dashboards/swarmprom-services-dash.json | \
      ssh $SSH_OPTIONS $SSH_USER@$1 'cat > ~/grafana/provisioning/dashboards/swarmprom-services-dash.json'
   cat ./grafana/provisioning/dashboards/swarmprom_dashboards.yml | \
      ssh $SSH_OPTIONS $SSH_USER@$1 'cat > ~/grafana/provisioning/dashboards/swarmprom_dashboards.yml'

  cat ./grafana/provisioning/datasources/datasource.yml | \
      sed "s/{{.AppNode}}/${hosts[$count]}/g" \
      | ssh $SSH_OPTIONS $SSH_USER@$1 'cat > ~/grafana/provisioning/datasources/datasource.yml'

  cat ./grafana/grafana.ini | ssh $SSH_OPTIONS $SSH_USER@$1 'cat > ~/grafana/grafana.ini'


  cat ./prometheus/prometheus.yml | \
      sed "s/{{.DBNode1}}/${hosts[2]}/g; s/{{.DBNode2}}/${hosts[3]}/g; s/{{.DBNode3}}/${hosts[4]}/g; s/{{.AppNode1}}/${hosts[0]}/g; s/{{.AppNode2}}/${hosts[1]}/g; s/{{.Label}}/AppNode$((count+1))/g" \
      | ssh $SSH_OPTIONS $SSH_USER@$1 'cat > ~/prometheus/prometheus.yml'

  cat ./prometheus/alert.rules | ssh $SSH_OPTIONS $SSH_USER@$1 'cat > ~/prometheus/alert.rules'
  cat ./alertmanager/config.yml | ssh $SSH_OPTIONS $SSH_USER@$1 'cat > ~/alertmanager/config.yml'

   ssh $SSH_OPTIONS $SSH_USER@$1 "sudo mkdir -p /etc/node-exporter && sudo chown -R 65534:65534 /etc/node-exporter"

   NODE_ID=$(ssh $SSH_OPTIONS $SSH_USER@$1 "docker info --format '{{.Swarm.NodeID}}'")
   NODE_NAME=$(ssh $SSH_OPTIONS $SSH_USER@$1 "docker node inspect --format '{{.Description.Hostname}}' $NODE_ID")

   echo "node_meta{node_id=\"$NODE_ID\", container_label_com_docker_swarm_node_id=\"$NODE_ID\", node_name=\"$NODE_NAME\"} 1" | \
     ssh $SSH_OPTIONS $SSH_USER@$1 "sudo sh -c 'cat > /etc/node-exporter/node-meta.prom'"

  ssh $SSH_OPTIONS $SSH_USER@$1  "bash -s" << EOF 
  sudo chown -R 65534:65534 /etc/node-exporter/node-meta.prom;
  sudo chown -R 65534:65534 /prometheus;
  sudo chown -R 65534:65534 /alertmanager;
  sudo chown -R 472:472 /var/lib/grafana;
EOF

  count=$((count+1))

  shift
done

set +x

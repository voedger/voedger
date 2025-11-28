#!/usr/bin/env bash
#
# Copyright (c) 2023 Sigma-Soft, Ltd.
# @author Aleksei Ponomarev
#
# Prepare scylla node: create directory for scylla data files,
# and copy scylla config to host

set -Eeuo pipefail

set -x

if [[ $# -ne 5 ]]; then
  echo "Usage: $0 <AppNode1> <AppNode2> <DBNode1> <DBNode2> <DBNode3>"
  exit 1
fi

source ./utils.sh

SSH_USER=$LOGNAME

AppNode1=$1
AppNode2=$2
DBNode1=$3
DBNode2=$4
DBNode3=$5

hosts=("app-node-1" "app-node-2" "db-node-1" "db-node-2" "db-node-3")

# Function to update /etc/hosts on a remote host using SSH
update_hosts_file() {
  local host=$1
  local ip=$2
  local hr=$3
  # Check if the hostname already exists in /etc/hosts
  if utils_ssh "$SSH_USER@$ip" "sudo grep -qF '$hr' /etc/hosts"; then
      # If the hostname exists, replace the existing entry
      utils_ssh "$SSH_USER@$ip" "sudo sed -i -E 's/.*\b$hr\b.*$/$hr\t$host/' /etc/hosts"
  else
      # If the hostname doesn't exist, add the new record
      utils_ssh "$SSH_USER@$ip" "sudo bash -c 'echo -e \"$hr\t$host\" >> /etc/hosts'"
  fi

  # SSH command to execute on the remote host
  # ssh $SSH_OPTIONS $SSH_USER@$ip "sudo bash -c 'echo -e \"$hr\t$host\" >> /etc/hosts'"
}

update_hosts_file_refactored() {
  local host=$1
  local ip=$2
  local hr=$3

  # SSH command to execute on the remote host
  utils_ssh "$SSH_USER@$ip" "sudo bash -c '
    if grep -qF \"$hr\" /etc/hosts; then
      sed -i -E \"s/.*\b$hr\b.*$/$hr\t$host/\" /etc/hosts
    else
      echo -e \"$hr\t$host\" >> /etc/hosts
    fi
  '"
}

args_array=("$@")
# Prepare for name resolving - iterate over each hostname and update /etc/hosts on each host
#i=0
#for host in "${hosts[@]}"; do
#  ip=${args_array[i]}
  # Iterate over the three IP addresses
#  for ip_address in "$@"; do
#      update_hosts_file $host $ip_address $ip
#  done
#((++i))
#done

count=0

while [ $# -gt 0 ] && [ $count -lt 2 ]; do
  echo "Processing: $1"
  utils_ssh "$SSH_USER@$1" "bash -s" << EOF
  sudo mkdir -p /prometheus && mkdir -p ~/prometheus;
  sudo mkdir -p /alertmanager && mkdir -p ~/alertmanager;
  mkdir -p ~/grafana/provisioning/dashboards;
  mkdir -p ~/grafana/provisioning/datasources;
  sudo mkdir -p /var/lib/grafana;
EOF

   cat ./grafana/provisioning/dashboards/docker-swarm-nodes.json | \
      utils_ssh "$SSH_USER@$1" 'cat > ~/grafana/provisioning/dashboards/docker-swarm-nodes.json'
   cat ./grafana/provisioning/dashboards/prometheus.json | \
      utils_ssh "$SSH_USER@$1" 'cat > ~/grafana/provisioning/dashboards/prometheus.json'
   cat ./grafana/provisioning/dashboards/docker-swarm-services.json | \
      utils_ssh "$SSH_USER@$1" 'cat > ~/grafana/provisioning/dashboards/docker-swarm-services.json'
   cat ./grafana/provisioning/dashboards/app-processors.json | \
      utils_ssh "$SSH_USER@$1" 'cat > ~/grafana/provisioning/dashboards/app-processors.json'
   cat ./grafana/provisioning/dashboards/dashboards.yml | \
      utils_ssh "$SSH_USER@$1" 'cat > ~/grafana/provisioning/dashboards/dashboards.yml'

  cat ./grafana/provisioning/datasources/datasource.yml | \
      sed "s/{{.AppNode}}/${hosts[$count]}/g" \
      | utils_ssh "$SSH_USER@$1" 'cat > ~/grafana/provisioning/datasources/datasource.yml'

  cat ./grafana/grafana.ini | utils_ssh "$SSH_USER@$1" 'cat > ~/grafana/grafana.ini'


  if [[ "${VOEDGER_EDITION:-}" == "SE3" ]]; then
      cat ./prometheus/prometheus.yml | \
          sed "s/{{.DBNode1}}/${hosts[2]}/g; s/{{.DBNode2}}/${hosts[3]}/g; s/{{.DBNode3}}/${hosts[4]}/g; s/{{.AppNode1}}/${hosts[0]}/g; s/{{.AppNode2}}/${hosts[1]}/g; s/{{.Label}}/AppNode$((count+1))/g" | \
          yq e 'del(.scrape_configs[] | select(.job_name == "node-exporter").static_configs[].targets[] | select(test("^app-node.*:9100$")))' - | \
          utils_ssh "$SSH_USER@$1" 'cat > ~/prometheus/prometheus.yml'
  else
      cat ./prometheus/prometheus.yml | \
          sed "s/{{.DBNode1}}/${hosts[2]}/g; s/{{.DBNode2}}/${hosts[3]}/g; s/{{.DBNode3}}/${hosts[4]}/g; s/{{.AppNode1}}/${hosts[0]}/g; s/{{.AppNode2}}/${hosts[1]}/g; s/{{.Label}}/AppNode$((count+1))/g" \
          | utils_ssh "$SSH_USER@$1" 'cat > ~/prometheus/prometheus.yml'
  fi

  cat ./prometheus/web.yml | utils_ssh "$SSH_USER@$1" 'cat > ~/prometheus/web.yml'
  if utils_ssh "$SSH_USER@$1" "if [ ! -f $HOME/prometheus/alert.rules ]; then exit 0; else exit 1; fi"; then
      echo "$HOME/prometheus/alert.rules does not exist on the remote host. Creating it now.";
      cat ./prometheus/alert.rules | utils_ssh "$SSH_USER@$1" 'cat > ~/prometheus/alert.rules';
  else
      echo "$HOME/prometheus/alert.rules already exists on the remote host.";
  fi

  if utils_ssh "$SSH_USER@$1" "if [ ! -f $HOME/alertmanager/config.yml ]; then exit 0; else exit 1; fi"; then
      echo "$HOME/alertmanager/config.yml does not exist on the remote host. Creating it now.";
      cat ./alertmanager/config.yml | utils_ssh "$SSH_USER@$1" 'cat > ~/alertmanager/config.yml';
  else
      echo "$HOME/alertmanager/config.yml already exists on the remote host.";
  fi

  utils_ssh "$SSH_USER@$1" "sudo mkdir -p /etc/node-exporter && sudo chown -R 65534:65534 /etc/node-exporter"

   NODE_ID=$(utils_ssh "$SSH_USER@$1" "docker info --format '{{.Swarm.NodeID}}'")
   NODE_NAME=$(utils_ssh "$SSH_USER@$1" "docker node inspect --format '{{.Description.Hostname}}' $NODE_ID")

   echo "node_meta{node_id=\"$NODE_ID\", container_label_com_docker_swarm_node_id=\"$NODE_ID\", node_name=\"$NODE_NAME\"} 1" | \
     utils_ssh "$SSH_USER@$1" "sudo sh -c 'cat > /etc/node-exporter/node-meta.prom'"

  utils_ssh "$SSH_USER@$1"  "bash -s" << EOF
  sudo chown -R 65534:65534 /etc/node-exporter/node-meta.prom;
  sudo chown -R 65534:65534 /prometheus;
  sudo chown -R 65534:65534 /alertmanager;
  sudo chown -R 472:472 /var/lib/grafana;
EOF

  count=$((count+1))

  shift
done

set +x

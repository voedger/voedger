#!/usr/bin/env bash
#
# Copyright (c) 2024 Sigma-Soft, Ltd.
# @author Aleksei Ponomarev
#

set -euo pipefail

if [ "$#" -lt 1 ]; then
  echo "Usage: $0 <node ip address for monitoring setup>" >&2
  exit 1
fi

source ../utils.sh

NODE=$1
SSH_USER=$LOGNAME

echo "Setting up monitoring stack on remote host $NODE..."

# Create the monitoring setup script to run on the remote machine
script="\
sudo mkdir -p /prometheus && mkdir -p ~/prometheus;
sudo mkdir -p /alertmanager && mkdir -p ~/alertmanager;
mkdir -p ~/grafana/provisioning/dashboards;
mkdir -p ~/grafana/provisioning/datasources;
sudo mkdir -p /var/lib/grafana;
sudo chown -R 65534:65534 /prometheus;
sudo chown -R 65534:65534 /alertmanager;
sudo chown -R 472:472 /var/lib/grafana;
"

echo "Creating monitoring directories on remote host..."
utils_ssh_interactive "$SSH_USER@$NODE" "bash -s" << EOF
$script
EOF

echo "Copying monitoring configuration files to remote host..."

# Copy Grafana configuration files
utils_scp "grafana/grafana.ini" "$SSH_USER@$NODE:~/grafana/grafana.ini"

# Create Grafana datasource configuration
datasource_config='apiVersion: 1
datasources:
  - name: Prometheus
    type: prometheus
    access: proxy
    url: http://prometheus:9090
    isDefault: true
    editable: true'

# Copy datasource configuration to remote host
echo "$datasource_config" | utils_ssh "$SSH_USER@$NODE" "cat > ~/grafana/provisioning/datasources/datasource.yml"

# Copy Grafana dashboard provisioning files
utils_scp "grafana/provisioning/dashboards/dashboards.yml" "$SSH_USER@$NODE:~/grafana/provisioning/dashboards/dashboards.yml"
utils_scp "grafana/provisioning/dashboards/app-processors.json" "$SSH_USER@$NODE:~/grafana/provisioning/dashboards/app-processors.json"
utils_scp "grafana/provisioning/dashboards/node-exporter-full.json" "$SSH_USER@$NODE:~/grafana/provisioning/dashboards/node-exporter-full.json"
utils_scp "grafana/provisioning/dashboards/prometheus.json" "$SSH_USER@$NODE:~/grafana/provisioning/dashboards/prometheus.json"

# Copy Prometheus configuration files
utils_scp "prometheus/prometheus.yml" "$SSH_USER@$NODE:/tmp/prometheus.yml"
utils_ssh "$SSH_USER@$NODE" "export VOEDGER_CE_NODE=$NODE && envsubst < /tmp/prometheus.yml > ~/prometheus/prometheus.yml"
utils_scp "prometheus/web.yml" "$SSH_USER@$NODE:~/prometheus/web.yml"
utils_scp "prometheus/alert.rules" "$SSH_USER@$NODE:~/prometheus/alert.rules"

# Copy Alertmanager configuration
utils_scp "alertmanager/config.yml" "$SSH_USER@$NODE:~/alertmanager/config.yml"

echo "Monitoring stack setup completed on remote host."

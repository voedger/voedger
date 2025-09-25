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

echo "Setting up monitoring stack on host $NODE..."
echo "Creating monitoring directories on host..."
sudo mkdir -p /prometheus && mkdir -p ~/prometheus;
sudo mkdir -p /alertmanager && mkdir -p ~/alertmanager;
mkdir -p ~/grafana/provisioning/dashboards;
mkdir -p ~/grafana/provisioning/datasources;
sudo mkdir -p /var/lib/grafana;
sudo chown -R 65534:65534 /prometheus;
sudo chown -R 65534:65534 /alertmanager;
sudo chown -R 472:472 /var/lib/grafana;

echo "Copying monitoring configuration files to host..."
cp grafana/grafana.ini ~/grafana/grafana.ini
# Create Grafana datasource configuration
datasource_config='apiVersion: 1
datasources:
  - name: Prometheus
    type: prometheus
    access: proxy
    url: http://prometheus:9090
    isDefault: true
    editable: true'

# Copy datasource configuration to host
echo "$datasource_config" | cat > ~/grafana/provisioning/datasources/datasource.yml

# Copy Grafana dashboard provisioning files
cp grafana/provisioning/dashboards/dashboards.yml ~/grafana/provisioning/dashboards/dashboards.yml
cp grafana/provisioning/dashboards/app-processors.json ~/grafana/provisioning/dashboards/app-processors.json
cp grafana/provisioning/dashboards/node-exporter-full.json ~/grafana/provisioning/dashboards/node-exporter-full.json
cp grafana/provisioning/dashboards/prometheus.json ~/grafana/provisioning/dashboards/prometheus.json

# Copy Prometheus configuration files
cp prometheus/prometheus.yml /tmp/prometheus.yml
export VOEDGER_CE_NODE=$NODE && envsubst < /tmp/prometheus.yml > ~/prometheus/prometheus.yml
cp prometheus/web.yml ~/prometheus/web.yml
cp prometheus/alert.rules ~/prometheus/alert.rules

# Copy Alertmanager configuration
cp alertmanager/config.yml ~/alertmanager/config.yml
echo "Monitoring stack setup completed on host."

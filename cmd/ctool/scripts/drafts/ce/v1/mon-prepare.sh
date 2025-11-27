#!/usr/bin/env bash
#
# Copyright (c) 2024 Sigma-Soft, Ltd.
# @author Aleksei Ponomarev
#

set -Eeuo pipefail

set -x

sudo mkdir -p /prometheus && mkdir -p ~/prometheus
sudo mkdir -p /alertmanager && mkdir -p ~/alertmanager
mkdir -p ~/grafana/provisioning/dashboards
mkdir -p ~/grafana/provisioning/datasources
sudo mkdir -p /var/lib/grafana

cp ./grafana/provisioning/dashboards/* ~/grafana/provisioning/dashboards
envsubst < ./grafana/provisioning/datasources/datasource.yml > ~/grafana/provisioning/datasources/datasource.yml
cp ./grafana/grafana.ini ~/grafana/grafana.ini


envsubst < ./prometheus/prometheus.yml > ~/prometheus/prometheus.yml

cp ./prometheus/web.yml ~/prometheus/web.yml
cp ./prometheus/alert.rules ~/prometheus/alert.rules
cp ./alertmanager/config.yml ~/alertmanager/config.yml

sudo chown -R 65534:65534 /prometheus
sudo chown -R 65534:65534 /alertmanager
sudo chown -R 472:472 /var/lib/grafana

set +x

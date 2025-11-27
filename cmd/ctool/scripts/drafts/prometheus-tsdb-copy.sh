#!/usr/bin/env bash
#
# Copyright (c) 2023 Sigma-Soft, Ltd.
# @author Aleksei Ponomarev
#
# Copy prometheus database to another host
#
# By default, Prometheus stores its time series database (TSDB)
# data and snapshots in a subdirectory named within the
# directory specified by --storage.tsdb.path. Therefore,
# the snapshots are stored in the snapshots directory
# relative to the --storage.tsdb.path.
#
# In compose file Mon stack use /prometheus folder on host and
# map this folder to prometheus image. So, snapshot will be store
# in /prometheus/snapshots

set -Eeuo pipefail

# Check if both source and destination IP addresses are provided
if [[ "$#" -ne 2 ]]; then
  echo "Usage: ./prometheus-tsdb-copy.sh <src_ip> <dst_ip>"
  exit 1
fi

source ./utils.sh

SSH_USER=$LOGNAME

# Assign arguments to variables
src_ip=$1
dst_ip=$2

# Define the snapshot directory
snapshot_dir="/prometheus/snapshots"

# Generate a timestamp for the snapshot
timestamp=$(date +%Y%m%d%H%M%S)

# Generate the snapshot file name
snapshot_file="prometheus_snapshot_$timestamp.tar.gz"

dst_name=$(getent hosts "$dst_ip" | awk '{print $2}')

ssh-keyscan -p "$(utils_SSH_PORT)" -H "$dst_name" >> ~/.ssh/known_hosts

snapshot=$(curl -u voedger:voedger -X POST http://$src_ip:9090/api/v1/admin/tsdb/snapshot | jq -r '.data.name')
# Make the snapshot on source host
if [ -z $snapshot ]; then
  echo "Error make prometheus snapshot."
  exit 1
else
  echo "Success make prometheus snapshot."
fi

# Compress the snapshot on the source host
if utils_ssh "$SSH_USER@$src_ip" "tar -czvf ~/$snapshot.tar.gz -C $snapshot_dir $snapshot"; then
  echo "Success compress prometheus snapshot."
else
  echo "Error compress prometheus snapshot."
  exit 1
fi


# Copy the compressed snapshot to the destination host
utils_scp -3 $SSH_USER@$src_ip:~/$snapshot.tar.gz $SSH_USER@$dst_ip:~

utils_ssh "$SSH_USER@$dst_ip" "
  # Exit immediately if any command exits with a non-zero status
  set -Eeuo pipefail

  # Extract the snapshot on the destination host
  sudo mkdir -p $snapshot_dir && sudo tar -xzvf ~/$snapshot.tar.gz -C $snapshot_dir

  # Move the extracted snapshot to the appropriate Prometheus directory
  sudo mv $snapshot_dir/$snapshot/* /prometheus
  sudo chown -R 65534:65534 /prometheus
"

# Cleanup: remove the snapshot files from both hosts
utils_ssh "$SSH_USER@$src_ip" "rm -rf ~/$snapshot.tar.gz"
utils_ssh "$SSH_USER@$dst_ip" "rm -rf ~/$snapshot.tar.gz"

echo "Prometheus base copied successfully!"

live_app_host=$(getent hosts "$src_ip" | awk '{print $2}')
app_host_idx=$(echo "$live_app_host" | rev | cut -c 1)

docker service update MonDockerStack_prometheus"$app_host_idx" --force --quiet
docker service update MonDockerStack_alertmanager"$app_host_idx" --force --quiet
docker service update MonDockerStack_cadvisor"$app_host_idx" --force --quiet


exit 0
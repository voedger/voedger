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

set -euo pipefail

# Check if both source and destination IP addresses are provided
if [[ "$#" -ne 2 ]]; then
  echo "Usage: ./prometheus-tsdb-copy.sh <src_ip> <dst_ip>"
  exit 1
fi

SSH_USER=$LOGNAME
SSH_OPTIONS='-o UserKnownHostsFile=/dev/null -o StrictHostKeyChecking=no -o LogLevel=ERROR'

# Assign arguments to variables
src_ip=$1
dst_ip=$2

# Define the snapshot directory
snapshot_dir="/prometheus/snapshots"

# Generate a timestamp for the snapshot
timestamp=$(date +%Y%m%d%H%M%S)

# Generate the snapshot file name
snapshot_file="prometheus_snapshot_$timestamp.tar.gz"

snapshot=$(curl -X POST http://$src_ip:9090/api/v1/admin/tsdb/snapshot | jq -r '.data.name') 
# Make the snapshot on source host
if [ -z $snapshot ]; then
  echo "Error make prometheus snapshot."
  exit 1
else
  echo "Success make prometheus snapshot."
fi

# Compress the snapshot on the source host
if ssh $SSH_OPTIONS $SSH_USER@$src_ip "tar -czvf ~/$snapshot.tar.gz -C $snapshot_dir $snapshot; echo \$?"; then
  echo "Success compress prometheus snapshot."
else
  echo "Error compress prometheus snapshot."
  exit 1
fi


# Copy the compressed snapshot to the destination host
scp -3 $SSH_OPTIONS $SSH_USER@$src_ip:~/$snapshot.tar.gz $SSH_USER@$dst_ip:~

# Extract the snapshot on the destination host
sudo mkdir -p $snapshot_dir && sudo tar -xzvf ~/$snapshot.tar.gz -C $snapshot_dir

# Move the extracted snapshot to the appropriate Prometheus directory
sudo mv $snapshot_dir/$snapshot/* /prometheus
sudo chown -R 65534:65534 /prometheus


# Cleanup: remove the snapshot files from both hosts
ssh $SSH_OPTIONS $SSH_USER@$src_ip "rm -rf ~/$snapshot.tar.gz"
rm -f ~/$snapshot.tar.gz

echo "Prometheus base copied successfully!"

exit 0
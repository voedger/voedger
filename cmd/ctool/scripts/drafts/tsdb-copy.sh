#!/usr/bin/env bash
#
# Copyright (c) 2023 Sigma-Soft, Ltd.
# @author Aleksei Ponomarev
#
# Copy prometheus database to another host
#
# By default, Prometheus stores its time series database (TSDB) 
# data and snapshots in a subdirectory named data within the 
# directory specified by --storage.tsdb.path. Therefore, 
# the snapshots are stored in the data/snapshots directory 
# relative to the --storage.tsdb.path.
# In compose flie Mon stack use /prometheus folder on host and 
# map this folder to prometheus image. So, snapshot will be store 
# in /prometheus/data/snapshots

set -euo pipefail

# Check if both source and destination IP addresses are provided
if [[ "$#" -ne 2 ]]; then
  echo "Usage: ./tsdb-copy.sh <src_ip> <dst_ip>"
  exit 1
fi

SSH_USER=$LOGNAME
SSH_OPTIONS='-o UserKnownHostsFile=/dev/null -o StrictHostKeyChecking=no -o LogLevel=ERROR'

# Assign arguments to variables
src_ip=$1
dst_ip=$2

# Define the snapshot directory
  snapshot_dir="/prometheus/snapshots"

# Create the snapshot directory if it doesn't exist
# mkdir -p "$snapshot_dir"

# Generate a timestamp for the snapshot
timestamp=$(date +%Y%m%d%H%M%S)

# Generate the snapshot file name
snapshot_file="prometheus_snapshot_$timestamp.tar.gz"

# get prometheus container id from src host 
# container_id=$(docker ps -q --filter "name=MonDockerStack_prometheus")

snapshot=$(curl -X POST http://$src_ip:9090/api/v1/admin/tsdb/snapshot | jq -r '.data.name') 
# Make the snapshot on source host
if [ -z $snapshot ]; then
  echo "Error when take prometheus snapshot."
  exit 1
else
  echo "Success make prometheus snapshot."
fi

# Find the latest snapshot directory
#latest_snapshot=$(ssh $SSH_USER@$src_ip "ls -td $snapshot_dir/* | head -1")

# Check if a snapshot directory was found
#if [ -z "$latest_snapshot" ]; then
#  echo "No snapshot directories found in $snapshot_dir on the source host."
#  exit 1
#fi

#snapshot_basename=$(basename "$latest_snapshot")

# Compress the snapshot on the source host
ssh $SSH_OPTIONS $SSH_USER@$src_ip "tar -czvf ~/$snapshot.tar.gz -C $snapshot_dir $snapshot; echo \$?"


# Copy the compressed snapshot to the destination host
scp $SSH_OPTIONS $SSH_USER@$src_ip:~/$snapshot.tar.gz ~

# Extract the snapshot on the destination host
sudo mkdir -p $snapshot_dir && tar -xzvf ~/$snapshot.tar.gz -C $snapshot_dir

# Move the extracted snapshot to the appropriate Prometheus directory
mv $snapshot_dir/$snapshot /prometheus
sudo chown -R 65534:65534 /prometheus


# Cleanup: remove the snapshot files from both hosts
ssh $SSH_OPTIONS $SSH_USER@$src_ip "rm -rf ~/$snapshot.tar.gz"
rm -f ~/$snapshot.tar.gz

echo "Prometheus base copied successfully!"

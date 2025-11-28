#!/usr/bin/env bash
# Copyright (c) 2024 Sigma-Soft, Ltd.
# @author Aleksei Ponomarev
#
# Deploy distributed file system using GlusterFS

set -Eeuo pipefail
set +x

if [ -z "${VOEDGER_SSH_KEY:-}" ]; then
    echo "VOEDGER_SSH_KEY must be set with ssh key path. Exiting..."
    exit 1
fi

is_success() {
    local output="$1"

    if [[ "$output" =~ success ]]; then
        echo "Operation was successful."
    else
        echo "Operation failed."
        return 1
    fi
}

source ./utils.sh

hosts=("db-node-1" "db-node-2" "db-node-3")

main_node=${hosts[0]}
peer_nodes=("${hosts[@]:1}")

# Install gluster FS on all nodes
for host in "${hosts[@]}"; do
    utils_ssh "$LOGNAME@$host" "sudo mkdir -p /data/glusterfs/voedger-vol1/brick1 && sudo mkdir -p /mnt/dfs"
    utils_ssh "$LOGNAME@$host" "sudo apt install glusterfs-server -y && sudo sudo systemctl enable --now glusterd"
done

for node in "${peer_nodes[@]}"; do
    echo "Probing $node from $main_node..."
    utils_ssh "$LOGNAME@$main_node" "sudo gluster peer probe $node"
done

attempts=5

# wait until glusterd run on all cluster nodes
for (( i=0; i<attempts; i++ )); do
    echo "Checking the pool list, attempt $(($i + 1))..."

    # Fetch the pool list and count the number of peers with 'Connected' status
    connected_peers=$(utils_ssh "$LOGNAME@$main_node" "sudo gluster pool list | grep -c 'Connected'")
    echo "$connected_peers peers connected"

    # Check if the number of connected peers matches the number of peer nodes
    if [[ "$connected_peers" -eq "${#hosts[@]}" ]]; then
        echo "All nodes are successfully connected."
        break
    else
        echo "Expected ${#hosts[@]} connected nodes, but found $connected_peers. Retry in 3 seconds..."
        sleep 3
    fi
done

create_output=$(utils_ssh "$LOGNAME@$main_node" "sudo gluster volume create voedger-vol1 replica 3 db-node-{1..3}:/data/glusterfs/voedger-vol1/brick1/ force")
is_success "$create_output"

start_output=$(utils_ssh "$LOGNAME@$main_node" "sudo gluster volume start voedger-vol1")
is_success "$start_output"

volume_entry="localhost:/voedger-vol1 /mnt/dfs glusterfs defaults,_netdev 0 0"
match_pattern="localhost:/voedger-vol1 /mnt/dfs glusterfs"

for host in "${hosts[@]}"; do
    cmd=$(cat <<EOF
    # Remove existing entry if it exists
    grep -q '$match_pattern' /etc/fstab && sudo sed -i '\\|$match_pattern|d' /etc/fstab;
    # Append the new or updated entry
    echo "$volume_entry" | sudo tee -a /etc/fstab > /dev/null
EOF
    )
    utils_ssh "$LOGNAME@$host" "$cmd"
#    utils_ssh "$LOGNAME@$host" 'echo "localhost:/voedger-vol1 /mnt/dfs glusterfs defaults,_netdev 0 0" | sudo tee -a /etc/fstab > /dev/null'
    utils_ssh "$LOGNAME@$host" "sudo mount -t glusterfs -o transport=tcp  localhost:/voedger-vol1 /mnt/dfs"
#   or sudo mount -a ?
done


exit 0

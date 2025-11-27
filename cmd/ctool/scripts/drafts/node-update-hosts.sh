#!/usr/bin/env bash
#
# Copyright (c) 2023 Sigma-Soft, Ltd.
# @author Aleksei Ponomarev
#
# Upsert /etc/hosts file with cluster node record

set -Eeuo pipefail

set -x

if [ "$#" -lt 3 ]; then
  echo "Usage: $0 <node ip address> <ip> <name>  " >&2
  exit 1
fi

source ./utils.sh

SSH_USER=$LOGNAME


update_hosts_file() {
  local node=$1
  local ip=$2
  local name=$3

  utils_ssh "$SSH_USER@$node" "sudo bash -c '
    if grep -qF \"$name\" /etc/hosts; then
      sed -i -E \"s/.*\\b$name\\b.*\$/$ip\t$name/\" /etc/hosts
    else
      echo -e \"$ip\t$name\" >> /etc/hosts
    fi
  '"
}

update_hosts_file "$@"

set +x

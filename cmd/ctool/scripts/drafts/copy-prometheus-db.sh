#!/usr/bin/env bash
#
# Copyright (c) 2023 Sigma-Soft, Ltd.
# @author Dmitry Molchanovsky
#

set -euo pipefail

set -x

if [ $# -ne 4 ]; then
  echo "Usage: $0 <source service> <source host> <dest service> <destination host>"
  exit 1
fi

SOURCE_SERVICE=$1
SOURCE_HOST=$2
DEST_SERVICE=$3
DEST_HOST=$4
SSH_USER=$LOGNAME

SSH_OPTIONS='-o UserKnownHostsFile=/dev/null -o StrictHostKeyChecking=no -o LogLevel=ERROR'

echo "Copy prometheus data base from $SOURCE_HOST to $DEST_HOST"

ssh $SSH_OPTIONS $SSH_USER@$SOURCE_HOST "docker service scale $SOURCE_SERVICE=0"
rsync -avz $SSH_USER@$SOURCE_HOST:/prometheus /tmp
ssh $SSH_OPTIONS $SSH_USER@$SOURCE_HOST "docker service scale $SOURCE_SERVICE=1"

ssh $SSH_OPTIONS $SSH_USER@$DEST_HOST "docker service scale $DEST_SERVICE=0"
rsync -avz /tmp/prometheus $SSH_USER@$DEST_HOST
ssh $SSH_OPTIONS $SSH_USER@$DEST_HOST "docker service scale $DEST_SERVICE=1"

set +x
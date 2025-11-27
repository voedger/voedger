#!/usr/bin/env bash
#
# Copyright (c) 2023 Sigma-Soft, Ltd.
# @author Dmitry Molchanovsky
#
#
# copies the ctool and ssh key file to the remote host

set -Eeuo pipefail

set -x

if [ $# -ne 3 ]; then
  echo "Usage: $0 <ctool-file> <key-file> <IP-address>"
  exit 1
fi

source ./utils.sh

SOURCE_CTOOL_FILE="$1"
SOURCE_KEY_FILE="$2"
REMOTE_HOST="$3"

SSH_USER=$LOGNAME

# Get the remote user's home directory, not the local one
REMOTE_HOME=$(utils_ssh "$SSH_USER@$REMOTE_HOST" "echo \$HOME" 2>/dev/null | tr -d '\r\n')
DEST_PATH="${REMOTE_HOME}/ctool"
DEST_CTOOL_FILE="${DEST_PATH}/ctool"
DEST_KEY_FILE="${DEST_PATH}/pkey"

utils_ssh "$SSH_USER@$REMOTE_HOST" "mkdir -p \"$DEST_PATH\""

if utils_ssh "$SSH_USER@$REMOTE_HOST" "[ -e \"$DEST_KEY_FILE\" ]"; then
  utils_ssh "$SSH_USER@$REMOTE_HOST" "chmod u+w \"$DEST_KEY_FILE\" && rm -f \"$DEST_KEY_FILE\""
fi

if utils_ssh "$SSH_USER@$REMOTE_HOST" "[ -e \"$DEST_CTOOL_FILE\" ]"; then
  utils_ssh "$SSH_USER@$REMOTE_HOST" "chmod u+w \"$DEST_CTOOL_FILE\" && rm -f \"$DEST_CTOOL_FILE\""
fi

utils_scp "$SOURCE_CTOOL_FILE" "$SSH_USER@$REMOTE_HOST:$DEST_CTOOL_FILE"
utils_scp "$SOURCE_KEY_FILE" "$SSH_USER@$REMOTE_HOST:$DEST_KEY_FILE"

set +x
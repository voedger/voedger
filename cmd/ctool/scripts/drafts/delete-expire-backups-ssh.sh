#!/usr/bin/env bash
#
# Copyright (c) 2024 Sigma-Soft, Ltd.
# @author Dmitry Molchanovsky
#
# writes a database backup task to cron
# over an ssh connection
set -Eeuo pipefail
set -x

if [ $# -ne 3 ] && [ $# -ne 3 ]; then
  echo "Usage: $0 <host address> <backups folder> <expire period>"
  exit 1
fi

source ./utils.sh

HOST=$1
FOLDER=$2
PERIOD=$3
SSH_USER=$LOGNAME

REMOTE_COMMAND="bash -s ${FOLDER} ${PERIOD}"

utils_ssh -t "${SSH_USER}"@"${HOST}" "${REMOTE_COMMAND}" < delete-expire-backups.sh

set +x
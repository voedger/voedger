#!/usr/bin/env bash
#
# Copyright (c) 2024 Sigma-Soft, Ltd.
# @author Dmitry Molchanovsky
# 
# writes a database backup task to cron
set -euo pipefail
set -x

if [ $# -ne 2 ]; then
  echo "Usage: $0 <cron schedule time> <ssh port>"  
  exit 1
fi

SCHEDULE=$1
SSH_PORT=$2
SSH_USER=$LOGNAME
CTOOL_PATH="/home/${SSH_USER}/ctool/ctool"
KEY_PATH="/home/${SSH_USER}/ctool/pkey"
CRON_HOST_NAME="app-node-1"
CRON_HOST=$(nslookup ${CRON_HOST_NAME} | awk '/^Address: / { print $2 }')
DB_NODE_1_HOST=$(nslookup "db-node-1" | awk '/^Address: / { print $2 }')
DB_NODE_2_HOST=$(nslookup "db-node-2" | awk '/^Address: / { print $2 }')
DB_NODE_3_HOST=$(nslookup "db-node-3" | awk '/^Address: / { print $2 }')
CURRENT_TIME=$(date "+%Y%m%d_%H%M%S")
BACKUP_FOLDER="/mnt/backup/voedger/\$(date +\%Y\%m\%d\%H\%M\%S)-backup"

set_cron_schedule(){
    CRON_FILE=$(mktemp)
    
    if crontab -l | grep -v "backup node"; then
      crontab -l | grep -v "backup node" > "${CRON_FILE}"
    fi

    echo "${SCHEDULE} ${CTOOL_PATH} backup node ${DB_NODE_1_HOST} ${BACKUP_FOLDER} ${KEY_PATH} --ssh-port ${SSH_PORT};${CTOOL_PATH} backup node ${DB_NODE_2_HOST} ${BACKUP_FOLDER} ${KEY_PATH} --ssh-port ${SSH_PORT};${CTOOL_PATH} backup node ${DB_NODE_3_HOST} ${BACKUP_FOLDER} ${KEY_PATH} --ssh-port ${SSH_PORT}" >> "${CRON_FILE}"
    echo "Modified cron file:"
    cat "${CRON_FILE}"
    crontab "${CRON_FILE}"
    echo "Cron schedule set successfully"
    rm "${CRON_FILE}"
}

set_cron_schedule

set +x
#!/usr/bin/env bash
#
# Copyright (c) 2024 Sigma-Soft, Ltd.
# @author Dmitry Molchanovsky
# 
# 
# displays a list of available backups on three DBNodes
set -euo pipefail

if [ $# -ne 0 ]; then
  echo "Usage: $0"
  exit 1
fi

BACKUP_FOLDER="/mnt/backup/voedger/"
HOST1="db-node-1"
HOST2="db-node-2"
HOST3="db-node-3"

HOST1_BACKUP_NAMES=$(ssh ubuntu@${HOST1} "find ${BACKUP_FOLDER} -mindepth 1 -maxdepth 1 -type d -printf '%f\n' 2>/dev/null" | sort -u)
HOST2_BACKUP_NAMES=$(ssh ubuntu@${HOST2} "find ${BACKUP_FOLDER} -mindepth 1 -maxdepth 1 -type d -printf '%f\n' 2>/dev/null" | sort -u)
HOST3_BACKUP_NAMES=$(ssh ubuntu@${HOST3} "find ${BACKUP_FOLDER} -mindepth 1 -maxdepth 1 -type d -printf '%f\n' 2>/dev/null" | sort -u)

NAMES="${HOST1_BACKUP_NAMES}"$'\n'"${HOST2_BACKUP_NAMES}"$'\n'"${HOST3_BACKUP_NAMES}"

BACKUP_NAMES=$(echo "${NAMES}" | sort -u -r)

if [ "${BACKUP_NAMES}" == "" ]; then
    {
      echo "No backups found"
    } > backups.lst
    exit 1
fi

{
echo "Backup                 |  DBNodes"
echo "-------------------------------------------------------"
for BACKUP_NAME in ${BACKUP_NAMES}; do
    HOSTS=""
    if echo "${HOST1_BACKUP_NAMES}" | grep -q "${BACKUP_NAME}"; then
        HOSTS+="${HOST1} "
    fi
    if echo "${HOST2_BACKUP_NAMES}" | grep -q "${BACKUP_NAME}"; then
        HOSTS+="${HOST2} "
    fi
    if echo "${HOST3_BACKUP_NAMES}" | grep -q "${BACKUP_NAME}"; then
        HOSTS+="${HOST3}"
    fi
    
    echo "${BACKUP_NAME}  |  ${HOSTS}"
    
done
} > backups.lst
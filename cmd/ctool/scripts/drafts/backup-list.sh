#!/usr/bin/env bash
#
# Copyright (c) 2024 Sigma-Soft, Ltd.
# @author Dmitry Molchanovsky
#
#
# displays a list of available backups on three DBNodes
set -Eeuo pipefail

if [ $# -gt 1 ]; then
  echo "Usage: $0 [<json>]"
  exit 1
fi

if [ $# -eq 1 ]; then
    OUTPUT_FORMAT=$1
else
    OUTPUT_FORMAT=""
fi
source ./utils.sh

BACKUP_FOLDER="/mnt/backup/voedger/"
HOST1="db-node-1"
HOST2="db-node-2"
HOST3="db-node-3"
SSH_USER=$LOGNAME

HOST1_BACKUP_NAMES=$(utils_ssh ${SSH_USER}@${HOST1} "find ${BACKUP_FOLDER} -mindepth 1 -maxdepth 1 -type d -printf '%f\n' 2>/dev/null" | sort -u)
HOST2_BACKUP_NAMES=$(utils_ssh ${SSH_USER}@${HOST2} "find ${BACKUP_FOLDER} -mindepth 1 -maxdepth 1 -type d -printf '%f\n' 2>/dev/null" | sort -u)
HOST3_BACKUP_NAMES=$(utils_ssh ${SSH_USER}@${HOST3} "find ${BACKUP_FOLDER} -mindepth 1 -maxdepth 1 -type d -printf '%f\n' 2>/dev/null" | sort -u)

NAMES="${HOST1_BACKUP_NAMES}"$'\n'"${HOST2_BACKUP_NAMES}"$'\n'"${HOST3_BACKUP_NAMES}"

BACKUP_NAMES=$(echo "${NAMES}" | sort -u -r)

if [ "${BACKUP_NAMES}" == "" ]; then
    {
      echo "No backups found"
    } > backups.lst
    exit 0
fi

if [ "${OUTPUT_FORMAT}" == "json" ]; then
    {
        echo "["
        for BACKUP_NAME in ${BACKUP_NAMES}; do
            HOSTS=""
            if echo "${HOST1_BACKUP_NAMES}" | grep -q "${BACKUP_NAME}"; then
                HOSTS+="\"${HOST1}\", "
            fi
            if echo "${HOST2_BACKUP_NAMES}" | grep -q "${BACKUP_NAME}"; then
                HOSTS+="\"${HOST2}\", "
            fi
            if echo "${HOST3_BACKUP_NAMES}" | grep -q "${BACKUP_NAME}"; then
                HOSTS+="\"${HOST3}\""
            fi
            echo "  {"
            echo "    \"Backup\": \"${BACKUP_NAME}\","
            echo "    \"DBNodes\": [${HOSTS}]"
            echo "  },"
        done
        echo "]"
    } > backups.lst
else
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
fi
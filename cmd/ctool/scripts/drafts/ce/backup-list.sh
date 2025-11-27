#!/usr/bin/env bash
#
# Copyright (c) 2024 Sigma-Soft, Ltd.
# @author Dmitry Molchanovsky
#
# Displays a list of available backups on three DBNodes

set -Eeuo pipefail
set -x

if [ $# -gt 1 ]; then
  echo "Usage: $0 [<json>]"
  exit 1
fi

if [ $# -eq 1 ]; then
    OUTPUT_FORMAT=$1
else
    OUTPUT_FORMAT=""
fi

BACKUP_FOLDER="/mnt/backup/voedger/"

BACKUP_NAMES=$(find "${BACKUP_FOLDER}" -mindepth 1 -maxdepth 1 -type d -printf '%f\n' 2>/dev/null | sort -u)

if [ -z "${BACKUP_NAMES}" ]; then
    {
      echo "No backups found"
    } > ../backups.lst
    exit 0
fi

if [ "${OUTPUT_FORMAT}" == "json" ]; then
    {
        echo "["
        FIRST=true
        for BACKUP_NAME in ${BACKUP_NAMES}; do
            if [ "${FIRST}" = true ]; then
                FIRST=false
            else
                echo ","
            fi
            echo "  {"
            echo "    \"Backup\": \"${BACKUP_NAME}\""
            echo "  }"
        done
        echo ""
        echo "]"
    } > ../backups.lst
else
    {
        echo "Backup"
        echo "-------------------------------------------------------"
        for BACKUP_NAME in ${BACKUP_NAMES}; do
            echo "${BACKUP_NAME}"
        done
    } > ../backups.lst
fi

set +x
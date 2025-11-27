#!/usr/bin/env bash
#
# Copyright (c) 2024 Sigma-Soft, Ltd.
# @author Dmitry Molchanovsky
#
# writes a database backup task to cron
set -Eeuo pipefail
set -x

if [ $# -ne 1 ] && [ $# -ne 2 ]; then
  echo "Usage: $0 <cron schedule time> [<expire time>]"
  exit 1
fi

if [ $# -eq 2 ]; then
    EXPIRE="--expire $2"
else
    EXPIRE=""
fi

SCHEDULE=$1

CTOOL_PATH="~/ctool/ctool"

CURRENT_TIME=$(date "+%Y%m%d_%H%M%S")
BACKUP_FOLDER="/mnt/backup/voedger/\$(date +\%Y\%m\%d\%H\%M\%S)-backup"

set_cron_schedule(){
    CRON_FILE=$(mktemp)

    # Создание временного cron файла без строки с "backup node"
    crontab -l | grep -v "backup node" > "${CRON_FILE}" || true

    # Добавление новой задачи в cron
    echo "${SCHEDULE} BACKUP_FOLDER=${BACKUP_FOLDER}; ${CTOOL_PATH} backup node \${BACKUP_FOLDER} ${EXPIRE}" >> "${CRON_FILE}"

    echo "Modified cron file:"
    cat "${CRON_FILE}"

    # Установка нового cron файла
    crontab "${CRON_FILE}"
    echo "Cron schedule set successfully"
    rm "${CRON_FILE}"
}

set_cron_schedule

set +x

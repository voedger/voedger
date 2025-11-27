#!/usr/bin/env bash
#
# Copyright (c) 2024 Sigma-Soft, Ltd.
# @author Dmitry Molchanovsky
#
#
# Removing backups with an expired shelf life
set -Eeuo pipefail
set -x
if [ $# -ne 2 ]; then
  echo "Usage: $0 <backups folder> <expire period>"
  exit 1
fi

FOLDER=$1
PERIOD=$2

# We determine the unit of measurement of the period (D - days, m - months)
UNIT=${PERIOD: -1}

# Delete the symbol of the measurement unit from the period
PERIOD=${PERIOD::-1}

# Calculate the date from which you need to remove folders
if [ "${UNIT}" == "d" ]; then
  DELETE_DATE=$(date -d "-${PERIOD} days" +%Y%m%d%H%M%S)
elif [ "${UNIT}" == "m" ]; then
  DELETE_DATE=$(date -d "-${PERIOD} months" +%Y%m%d%H%M%S)
else
  echo "The wrong time format.Use the number with the symbol 'd' or 'm' (for example, 20d or 3m)."
  exit 1
fi

cd ${FOLDER}

for DIR in $(ls -d *"-backup"/); do
  DIR_NAME=$(basename ${DIR})
  DIR_DATE=$(echo ${DIR_NAME} | cut -f1 -d'-')
  if [[ ${DIR_DATE} =~ ^[0-9]{14}$ ]] && [ "${DIR_DATE}" -lt "${DELETE_DATE}" ]; then
    echo "Delete backup ${DIR_NAME}"
    rm -rf ${FOLDER}${DIR_NAME}
  fi
done

set +x
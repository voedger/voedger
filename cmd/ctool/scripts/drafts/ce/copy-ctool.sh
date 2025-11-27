#!/usr/bin/env bash
#
# Copyright (c) 2024 Sigma-Soft, Ltd.
# @author Dmitry Molchanovsky
#
#
# copies the ctool and cluster.json files to the ~/ctool/

set -Eeuo pipefail

set -x

if [ $# -ne 1 ]; then
  echo "Usage: $0 <source-path>"
  exit 1
fi

SOURCE_PATH=$1
if [[ "${SOURCE_PATH}" != */ ]]; then
    SOURCE_PATH="${SOURCE_PATH}/"
fi

SOURCE_CTOOL_FILE="${SOURCE_PATH}ctool"
SOURCE_CONF_FILE="${SOURCE_PATH}cluster.json"

DEST_PATH="${HOME}/ctool"
DEST_CTOOL_FILE="${DEST_PATH}/ctool"
DEST_CONF_FILE="${DEST_PATH}/cluster.json"

mkdir -p "${DEST_PATH}"

if [ -e "$DEST_CTOOL_FILE" ]; then
  chmod u+w "$DEST_CTOOL_FILE" && rm -f "$DEST_CTOOL_FILE"
fi

if [ -e "$DEST_CONF_FILE" ]; then
  chmod u+w "$DEST_CONF_FILE" && rm -f "$DEST_CONF_FILE"
fi

cp ${SOURCE_CTOOL_FILE} ${DEST_CTOOL_FILE}
cp ${SOURCE_CONF_FILE} ${DEST_CONF_FILE}

set +x
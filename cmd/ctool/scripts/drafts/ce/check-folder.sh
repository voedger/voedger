#!/usr/bin/env bash
#
# Copyright (c) 2024 Sigma-Soft, Ltd.
# @author Dmitry Molchanovsky
#
#
# Verification of the availability of a folder on the local machine and access rights to it

set -Eeuo pipefail

set -x

if [ $# -ne 1 ]; then
  echo "Usage: $0 <folder path>"
  exit 1
fi


FOLDER_PATH=$1
USER=$LOGNAME

if [ -d "${FOLDER_PATH}" ]; then
  echo "Folder ${FOLDER_PATH} exists on the local machine."
  if [ -r "${FOLDER_PATH}" ] && [ -w "${FOLDER_PATH}" ]; then
    echo "User ${USER} has the rights to read/write in folder ${FOLDER_PATH}."
  else
    echo "The user ${USER} does not have rights to read/write in the folder ${FOLDER_PATH}."
    exit 2
  fi
else
  echo "The folder ${FOLDER_PATH} does not exist on the local machine."
  exit 1
fi

set +x

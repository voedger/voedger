#!/usr/bin/env bash
#
# Copyright (c) 2024 Sigma-Soft, Ltd.
# @author Dmitry Molchanovsky
#
#
# Verification of the availability of a folder on a remote host and access rights to it

set -Eeuo pipefail

set -x

if [ $# -ne 2 ]; then
  echo "Usage: $0 <remote host> <folder path>"
  exit 1
fi

source ./utils.sh

REMOTE_HOST=$1
FOLDER_PATH=$2
SSH_USER=$LOGNAME


utils_ssh ${SSH_USER}@${REMOTE_HOST} "if [ -d ${FOLDER_PATH} ]; then
                    echo 'Folder ${FOLDER_PATH} exists on a remote host.'
                    if [ -r ${FOLDER_PATH} ] && [ -w ${FOLDER_PATH} ]; then
                        echo 'User ${SSH_USER} has the rights to read/write in folder ${FOLDER_PATH}.'
                    else
                        echo 'The user ${SSH_USER} does not have rights to read/write in the folder ${FOLDER_PATH}.'
                        exit 2
                    fi
                else
                    echo 'The folder ${FOLDER_PATH} does not exist on the remote host ${REMOTE_HOST}.'
                    exit 1
                fi"

set +x


#!/usr/bin/env bash
#
# Copyright (c) 2023 Sigma-Soft, Ltd.
# @author Dmitry Molchanovsky
#
# Checks the availability of the host and its compliance with the minimum hardware requirements
#
set -euo pipefail

if [ $# -lt 2 ]; then
  echo "Usage: $0 <remote host IP> <minimum RAM MB>"
  exit 1
fi

# Assign the arguments to variables
REMOTE_HOST=$1
MIN_RAM=$2

SSH_USER=$LOGNAME
SSH_OPTIONS='-o UserKnownHostsFile=/dev/null -o StrictHostKeyChecking=no -o LogLevel=ERROR'


# Check if minimum RAM is set to 0
if [ "$MIN_RAM" -eq 0 ]; then
  ssh $SSH_OPTIONS $SSH_USER@$REMOTE_HOST exit
  ssh_exit_status=$?

  if [ $ssh_exit_status -eq 0 ]; then
    echo "SSH connection established with host $REMOTE_HOST."
  else
    echo "Failed to establish SSH connection with host $remote_host."
    exit 1  # Exit the script with an error
  fi
  # Check SSH connection
  echo "Skipping RAM check."
else
  # Establish SSH connection with the remote host and retrieve RAM information
  ram=$(ssh $SSH_OPTIONS $SSH_USER@$REMOTE_HOST free -m 2>/dev/null | awk 'NR==2{print $2}')

  # Check the exit status of the ssh command
  ssh_exit_status=$?

  if [ $ssh_exit_status -eq 0 ]; then
    echo "SSH connection established with host $REMOTE_HOST."
    # Compare RAM size with the minimum requirement
    if [ $ram -ge $MIN_RAM ]; then
      echo "RAM size on host $REMOTE_HOST ($ram MB) is sufficient."
      exit 0  # Exit the script without an error
    else
      echo "RAM size on host $REMOTE_HOST ($ram MB) is insufficient. Minimum requirement: $MIN_RAM MB."
      exit 1  # Exit the script with an error
    fi
  else
    echo "Failed to establish SSH connection with host $REMOTE_HOST."
    exit 1  # Exit the script with an error
  fi
fi


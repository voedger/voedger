#!/usr/bin/env bash
# Copyright (c) 2023 Sigma-Soft, Ltd.
# @author Aleksei Ponomarev

set -Eeuo pipefail

set -x

utils_SSH_PORT() {
    port="${VOEDGER_NODE_SSH_PORT:-22}"
    echo "$port"
}

utils_SSH_KEY() {
    key="${VOEDGER_SSH_KEY:-}"
    echo "$key"
}

utils_SSH_OPTS() {
    opts="-q -o UserKnownHostsFile=/dev/null -o StrictHostKeyChecking=no -o LogLevel=ERROR"
    echo "$opts"
}

utils_SCP_OPTS() {

    opts="-q -o UserKnownHostsFile=/dev/null -o StrictHostKeyChecking=no -o LogLevel=ERROR"

    if [ -n "${VOEDGER_NODE_SSH_PORT:-}" ]; then
        opts+=" -P ${VOEDGER_NODE_SSH_PORT}"
    fi

echo "$opts"
}

utils_ssh() {
  local ssh_options_string="$(utils_SSH_OPTS) -p $(utils_SSH_PORT) -i $(utils_SSH_KEY)"

  # Split the string into an array
  IFS=' ' read -r -a ssh_options <<< "$ssh_options_string"

  local ssh_result

  # Pass options as separate arguments
  ssh_result=$(ssh "${ssh_options[@]}" "$@")
  # Capture the exit status of the ssh command
  local ssh_exit_status=$?

  # Return the SSH command result
  echo "$ssh_result"

  return "$ssh_exit_status"
}

utils_scp() {
  local ssh_options_string="$(utils_SSH_OPTS) -P $(utils_SSH_PORT) -i $(utils_SSH_KEY)"

  IFS=' ' read -r -a ssh_options <<< "$ssh_options_string"
  scp "${ssh_options[@]}" "$@"
}

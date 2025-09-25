#!/usr/bin/env bash
#
# Setup passwordless sudo on remote Ubuntu server for CE deployment
#
set -euo pipefail

if [ $# -ne 1 ]; then
    echo "Usage: $0 <node>"
    exit 1
fi

source ../utils.sh

NODE=$1
SSH_USER=${SSH_USER:-$LOGNAME}

# Set default SSH key if not provided
if [ -z "${VOEDGER_SSH_KEY:-}" ]; then
    export VOEDGER_SSH_KEY="${HOME}/.ssh/id_rsa"
fi

echo "Setting up passwordless sudo for user '$SSH_USER' on CE node $NODE..."
echo "Configuring passwordless sudo..."
if [ -z "$SSH_USER_PASSWORD" ]; then
  utils_ssh "$SSH_USER@$NODE" "sudo -S bash -c 'touch /etc/sudoers.d/$SSH_USER | echo \"$SSH_USER ALL=(ALL) NOPASSWD:ALL\" | > sudo tee /etc/sudoers.d/$SSH_USER && sudo chmod 440 /etc/sudoers.d/$SSH_USER'"
else
  utils_ssh "$SSH_USER@$NODE" "echo '$SSH_USER_PASSWORD' | base64 -d | sudo -S bash -c 'touch /etc/sudoers.d/$SSH_USER | echo \"$SSH_USER ALL=(ALL) NOPASSWD:ALL\" | > sudo tee /etc/sudoers.d/$SSH_USER && sudo chmod 440 /etc/sudoers.d/$SSH_USER'"
fi
echo "Passwordless sudo configured successfully"

# Verify the setup worked
if utils_ssh "$SSH_USER@$NODE" "sudo -n true" 2>/dev/null; then
    echo "Passwordless sudo setup completed successfully for user '$SSH_USER' on CE node $NODE"
else
    echo "Failed to configure passwordless sudo for user '$SSH_USER' on CE node $NODE"
    exit 1
fi

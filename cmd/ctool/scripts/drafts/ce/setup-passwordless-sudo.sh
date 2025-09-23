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

# Check if passwordless sudo is already configured
echo "Checking if passwordless sudo is already configured..."
if utils_ssh "$SSH_USER@$NODE" "sudo -n true" 2>/dev/null; then
    echo "Passwordless sudo is already configured on CE node $NODE"
    exit 0
fi

echo "Passwordless sudo is not configured."
echo ""
echo "MANUAL SETUP REQUIRED:"
echo "Please run the following command in a separate terminal to configure passwordless sudo:"
echo ""
echo "ssh -t -i ${VOEDGER_SSH_KEY} $SSH_USER@$NODE \"echo '$SSH_USER ALL=(ALL) NOPASSWD:ALL' | sudo tee /etc/sudoers.d/$SSH_USER && sudo chmod 440 /etc/sudoers.d/$SSH_USER\""
echo ""
echo "After running the above command, press Enter to continue..."
read -p "Press Enter when passwordless sudo is configured: "

echo "Verifying passwordless sudo configuration..."

# Verify the setup worked
if utils_ssh "$SSH_USER@$NODE" "sudo -n true" 2>/dev/null; then
    echo "Passwordless sudo setup completed successfully on CE node $NODE"
else
    echo "Failed to configure passwordless sudo on CE node $NODE"
    exit 1
fi

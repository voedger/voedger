#!/usr/bin/env bash
#
# Setup passwordless sudo for CE deployment (runs locally on the target node)
#
set -Eeuo pipefail

SSH_USER=${SSH_USER:-$LOGNAME}

echo "Setting up passwordless sudo for user '$SSH_USER'..."

# Check if passwordless sudo is already configured
echo "Checking if passwordless sudo is already configured..."
if sudo -n true 2>/dev/null; then
    echo "Passwordless sudo is already configured for user '$SSH_USER'"
    exit 0
fi

echo "Configuring passwordless sudo..."
if [ -z "${SSH_USER_PASSWORD:-}" ]; then
    # No password provided, try direct sudo (assumes current session has sudo access)
    echo "$SSH_USER ALL=(ALL) NOPASSWD:ALL" | sudo tee /etc/sudoers.d/$SSH_USER > /dev/null
    sudo chmod 440 /etc/sudoers.d/$SSH_USER
else
    # Use provided password
    echo "$SSH_USER_PASSWORD" | base64 -d | sudo -S bash -c "echo \"$SSH_USER ALL=(ALL) NOPASSWD:ALL\" | tee /etc/sudoers.d/$SSH_USER > /dev/null && chmod 440 /etc/sudoers.d/$SSH_USER"
fi

echo "Passwordless sudo configured successfully"

# Verify the setup worked
if sudo -n true 2>/dev/null; then
    echo "Passwordless sudo setup completed successfully for user '$SSH_USER'"
else
    echo "Failed to configure passwordless sudo for user '$SSH_USER'"
    exit 1
fi

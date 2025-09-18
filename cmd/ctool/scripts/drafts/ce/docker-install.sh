#!/usr/bin/env bash
#
# Copyright (c) 2024 Sigma-Soft, Ltd.
# @author Aleksei Ponomarev
#
# Install docker to Voedger CE
#
set -euo pipefail

if [ "$#" -lt 1 ]; then
  echo "Usage: $0 <node ip address for docker install>" >&2
  exit 1
fi

source ../utils.sh

# This script runs on the local machine but needs to detect the remote Ubuntu version
# We'll detect it on the remote machine via SSH
NODE=$1
SSH_USER=$LOGNAME

# Detect Ubuntu version on the remote machine
echo "Detecting Ubuntu version on remote host $NODE..."
if utils_ssh "$SSH_USER@$NODE" 'command -v lsb_release >/dev/null 2>&1'; then
    release=$(utils_ssh "$SSH_USER@$NODE" 'lsb_release -rs')
elif utils_ssh "$SSH_USER@$NODE" '[ -f /etc/os-release ]'; then
    release=$(utils_ssh "$SSH_USER@$NODE" 'grep VERSION_ID /etc/os-release | cut -d"\"" -f2')
else
    echo "Cannot detect Ubuntu version on remote host. Installing lsb-release first..."
    utils_ssh "$SSH_USER@$NODE" 'sudo apt-get update && sudo apt-get install -y lsb-release'
    release=$(utils_ssh "$SSH_USER@$NODE" 'lsb_release -rs')
fi

echo "Detected Ubuntu version: $release"

if [[ $release == "20.04" ]]; then
    echo "This is Ubuntu 20.04"
    VERSION_STRING="5:20.10.23~3-0~ubuntu-focal"

elif [[ $release == "22.04" ]]; then
    echo "This is Ubuntu 22.04"
    VERSION_STRING="5:20.10.23~3-0~ubuntu-jammy"
elif [[ $release == "18.04" ]]; then
    echo "This is Ubuntu 18.04"
    VERSION_STRING="5:24.0.2-1~ubuntu.18.04~bionic"
else
    echo "This script only supports Ubuntu 20.04, 22.04 and 18.04"
    echo "Detected version: $release"
    exit 1
fi

script="\
        sudo apt-get update;
        sudo apt-get install -y lsb-release;
        sudo add-apt-repository ppa:rmescandon/yq -y;
        sudo apt-get update;
	sudo apt-get install \
		ca-certificates \
			curl \
			gnupg \
		yq jq netcat -y;

	curl -fsSL https://download.docker.com/linux/ubuntu/gpg | \
		sudo gpg --dearmor -o /usr/share/keyrings/docker-archive-keyring.gpg;

	echo \
		\"deb [arch=\$(dpkg --print-architecture) signed-by=/usr/share/keyrings/docker-archive-keyring.gpg] https://download.docker.com/linux/ubuntu \
		\$(lsb_release -cs) stable\" | sudo tee /etc/apt/sources.list.d/docker.list > /dev/null;

	sudo apt-get update;
	sudo apt-get install docker-ce=$VERSION_STRING docker-ce-cli=$VERSION_STRING containerd.io docker-compose -y;
	if ! getent group docker > /dev/null 2>&1; then
		sudo groupadd docker;
	fi;
	sudo usermod -aG docker ubuntu || true;
	sudo systemctl enable docker;
		sudo systemctl start docker; \
"

# Check if docker is installed and install if not
docker_ins=0
utils_ssh "$SSH_USER@$NODE" 'command docker -v &>/dev/null' || docker_ins=1
if [[ $docker_ins -eq 0 ]]; then
  echo "Docker is already installed on the remote host."
  echo "Starting Docker daemon on remote host..."
  utils_ssh "$SSH_USER@$NODE" "sudo systemctl start docker"
else
  echo "Docker is not installed on the host. Installing it now..."
  utils_ssh_interactive "$SSH_USER@$NODE" "bash -s" << EOF
  $script
EOF
fi

exit 0

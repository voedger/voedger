#!/usr/bin/env bash
#
# Copyright (c) 2023 Sigma-Soft, Ltd.
# @author Aleksei Ponomarev
#
# Install docker to dedicated nodes
#
set -Eeuo pipefail

if [ "$#" -lt 1 ]; then
  echo "Usage: $0 <node ip address for docker install>" >&2
  exit 1
fi

source ./utils.sh

NODE=$1
SSH_USER=$LOGNAME

# Get Ubuntu version from the remote host, not the local machine
release=$(utils_ssh "$SSH_USER@$NODE" "lsb_release -rs" 2>/dev/null | tr -d '\r\n' || echo "unknown")

if [[ $release == "20.04" ]]; then
	echo "This is Ubuntu 20.04"
	VERSION_STRING="5:20.10.23~3-0~ubuntu-focal"

elif [[ $release == "22.04" ]]; then
	echo "This is Ubuntu 22.04"
	VERSION_STRING="5:20.10.23~3-0~ubuntu-jammy"
else
	echo "This script only supports Ubuntu 20.04 and 22.04 (detected: $release)"
	exit 1
fi

script="\
        sudo add-apt-repository ppa:rmescandon/yq -y;
        sudo apt-get update;
	sudo apt-get install \
		ca-certificates \
			curl \
			gnupg \
		lsb-release \
		yq jq netcat -y;

	curl -fsSL https://download.docker.com/linux/ubuntu/gpg | \
		sudo gpg --dearmor -o /usr/share/keyrings/docker-archive-keyring.gpg;

	echo \
		\"deb [arch=\$(dpkg --print-architecture) signed-by=/usr/share/keyrings/docker-archive-keyring.gpg] https://download.docker.com/linux/ubuntu \
		\$(lsb_release -cs) stable\" | sudo tee /etc/apt/sources.list.d/docker.list > /dev/null;

	sudo apt-get update;
	sudo apt-get install docker-ce=$VERSION_STRING docker-ce-cli=$VERSION_STRING containerd.io -y;
	sudo groupadd docker;
	sudo usermod -aG docker ubuntu;
	sudo systemctl enable docker; \
"

# Check if docker is installed and install if not
docker_ins=0
utils_ssh "$SSH_USER@$NODE" 'command docker -v &>/dev/null' || docker_ins=1
if [[ $docker_ins -eq 0 ]]; then
  echo "Docker is already installed on the remote host."
else
  echo "Docker is not installed on the host. Installing it now..."
  utils_ssh "$SSH_USER@$NODE" "bash -s" << EOF
  $script
EOF
fi


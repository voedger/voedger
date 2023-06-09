#!/usr/bin/env bash
#
# Copyright (c) 2023 Sigma-Soft, Ltd.
# @author Aleksei Ponomarev
#
# Install docker to dedicated nodes
#
set -euo pipefail

if [ "$#" -lt 1 ]; then
  echo "Usage: $0 <node ip address for docker install>" >&2
  exit 1
fi

VERSION_STRING="5:20.10.23~3-0~ubuntu-focal"
NODE=$1
SSH_USER=$LOGNAME
SSH_OPTIONS='-o UserKnownHostsFile=/dev/null -o StrictHostKeyChecking=no -o LogLevel=ERROR'

script="\
        sudo add-apt-repository ppa:rmescandon/yq -y;
        sudo apt-get update;
	sudo apt-get install \
		ca-certificates \
			curl \
			gnupg \
		lsb-release \
		yq jq -y;

	curl -fsSL https://download.docker.com/linux/ubuntu/gpg | \
		sudo gpg --dearmor -o /usr/share/keyrings/docker-archive-keyring.gpg;

	echo \
		\"deb [arch=$(dpkg --print-architecture) signed-by=/usr/share/keyrings/docker-archive-keyring.gpg] https://download.docker.com/linux/ubuntu \
		$(lsb_release -cs) stable\" | sudo tee /etc/apt/sources.list.d/docker.list > /dev/null;

	sudo apt-get update;
	sudo apt-get install docker-ce=$VERSION_STRING docker-ce-cli=$VERSION_STRING containerd.io -y;
	sudo groupadd docker;
	sudo usermod -aG docker ubuntu;
	sudo systemctl enable docker; \
"

# Check if docker is installed and install if not
docker_ins=0
ssh $SSH_OPTIONS $SSH_USER@$NODE 'command docker -v &>/dev/null' || docker_ins=1
if [[ $docker_ins -eq 0 ]]; then
  echo "Docker is already installed on the remote host."
else
  echo "Docker is not installed on the host. Installing it now..."
  ssh $SSH_OPTIONS $SSH_USER@$NODE "bash -s" << EOF
  $script
EOF
fi


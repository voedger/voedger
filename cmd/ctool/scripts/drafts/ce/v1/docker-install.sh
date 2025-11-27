#!/usr/bin/env bash
#
# Copyright (c) 2024 Sigma-Soft, Ltd.
# @author Aleksei Ponomarev
#
# Install docker to Voedger CE
#
set -Eeuo pipefail

set -x

release=$(lsb_release -rs)

if [[ $release == "20.04" ]]; then
    echo "This is Ubuntu 20.04"
    VERSION_STRING="5:20.10.23~3-0~ubuntu-focal"

elif [[ $release == "22.04" ]]; then
    echo "This is Ubuntu 22.04"
    VERSION_STRING="5:20.10.23~3-0~ubuntu-jammy"
else
    echo "This script only supports Ubuntu 20.04 and 22.04"
    exit 1
fi

# Check if docker is installed and install if not
docker_ins=0
command docker -v &>/dev/null || docker_ins=1

if [[ $docker_ins -eq 0 ]]; then
    echo "Docker is already installed on the remote host."
else
    echo "Docker is not installed on the host. Installing it now..."
    sudo add-apt-repository ppa:rmescandon/yq -y
    sudo apt-get update
    sudo apt-get install ca-certificates curl gnupg lsb-release yq jq netcat -y

    curl -fsSL https://download.docker.com/linux/ubuntu/gpg | \
        sudo gpg --dearmor -o /usr/share/keyrings/docker-archive-keyring.gpg

    echo "deb [arch=$(dpkg --print-architecture) signed-by=/usr/share/keyrings/docker-archive-keyring.gpg] https://download.docker.com/linux/ubuntu $(lsb_release -cs) stable" | \
sudo tee /etc/apt/sources.list.d/docker.list > /dev/null

    sudo apt-get update
    sudo apt-get install docker-ce=$VERSION_STRING docker-ce-cli=$VERSION_STRING containerd.io docker-compose -y
    sudo groupadd docker
    sudo usermod -aG docker ubuntu
    sudo systemctl enable docker
fi

exit 0

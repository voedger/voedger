#!/usr/bin/env bash
#
# Copyright (c) 2023 Sigma-Soft, Ltd.
# @author Aleksei Ponomarev

set -euo pipefail

sudo add-apt-repository ppa:rmescandon/yq -y
sudo apt update 
sudo apt install yq -y
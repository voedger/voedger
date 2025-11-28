#!/usr/bin/env bash
#
# Copyright (c) 2024 Sigma-Soft, Ltd.
# @author Aleksei Ponomarev
#
# init Voedger CE
#
set -Eeuo pipefail

# install comman software components
./docker-install.sh

# start db
./db-start.sh

# Prepare mon components: copy prometheus, grafana configs, etc.
./mon-prepare.sh

# Start monitoring
./mon-start.sh

# Start app
./ce-start.sh

exit 0

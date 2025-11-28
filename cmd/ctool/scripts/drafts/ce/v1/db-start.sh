#!/usr/bin/env bash
#
# Copyright (c) 2024 Sigma-Soft, Ltd.
# @author Aleksei Ponomarev
#

set -Eeuo pipefail


if [ -n "${VOEDGER_CE_NODE:-}" ]; then
    envsubst < ./docker-compose-db.yml | sudo docker-compose -p DBDockerStack -f - up -d
else
   echo "Error deploy scylla db. Use export VOEDGER_CE_NODE= <hostname | ipaddress>."
   exit 1
fi

exit 0

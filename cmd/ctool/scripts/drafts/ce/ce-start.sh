#!/usr/bin/env bash
#
# Copyright (c) 2024 Sigma-Soft, Ltd.
# @author Aleksei Ponomarev
#
# envsubst < ./docker-compose.yml > docker-compose-tmp.yml
# docker-compose -p voedger -f ./docker-compose-tmp.yml ps

set -euo pipefail
set -x

if [ -n "${VOEDGER_CE_NODE:-}" ]; then
    sudo bash -c "echo \"${VOEDGER_CE_NODE} db-node-1\" >> /etc/hosts"
    envsubst < ./docker-compose-tmp.yml > ./docker-compose.yml
    sudo docker-compose -p voedger  up -d
else
   echo "Error deploy Voedger CE. Use export VOEDGER_CE_NODE= <hostname | ipaddress>."
   exit 1
fi

set +x
exit 0
                                                                                                             
                                                                                                             

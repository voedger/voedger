#!/usr/bin/env bash
#
# Copyright (c) 2024 Sigma-Soft, Ltd.
# @author Aleksei Ponomarev
#
# envsubst < ./docker-compose.yml > docker-compose-tmp.yml
# docker-compose -p voedger -f ./docker-compose-tmp.yml ps

set -Eeuo pipefail
set -x

HOSTNAME="db-node-1"

if [ -n "${VOEDGER_CE_NODE:-}" ]; then
    # Check if the record with db-node-1 is present
    if grep -q "$HOSTNAME" /etc/hosts; then
        # If present, update the record with the new IP
        sudo sed -i".bak" "/$HOSTNAME/c\\$VOEDGER_CE_NODE $HOSTNAME" /etc/hosts
    else
        # If not present, add the new record
        sudo bash -c "echo \"$VOEDGER_CE_NODE $HOSTNAME\" >> /etc/hosts"
    fi
    envsubst < ./docker-compose-tmp.yml > ./docker-compose.yml
    sudo docker-compose -p voedger  up -d
else
   echo "Error deploy Voedger CE. Use export VOEDGER_CE_NODE= <hostname | ipaddress>."
   exit 1
fi

set +x
exit 0



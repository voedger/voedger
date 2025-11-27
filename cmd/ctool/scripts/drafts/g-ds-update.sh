#!/usr/bin/env bash
#
# Copyright (c) 2024 Sigma-Soft, Ltd.
# @author Aleksei Ponomarev
#
# Add new user to grafana

set -Eeuo pipefail

set -x

if [[ $# -ne 4 ]]; then
  echo "Usage: $0 <app-node> <grafana admin user> <grafana admin password> <new-user-password>"
  exit 1
fi

HOST=$1
# Admin credentials
ADMIN_USER=$2
ADMIN_PASSWORD=$3
DATASOURCE_NAME="Prometheus"
NEW_BASIC_AUTH_USER="voedger"
NEW_BASIC_AUTH_PASSWORD="$4"

DsIDbyName() {
  local name="$1"
  local resp
  local url="http://$HOST:3000/api/datasources/name/$name"

  resp=$(curl -s -u "$ADMIN_USER:$ADMIN_PASSWORD" "$url")
  jq '.id' <<< "$resp"
}

dsID=$(DsIDbyName "${DATASOURCE_NAME}")

UPDATE_DS_PAYLOAD=$(cat <<-EOF
  {
    "id": ${dsID},
    "name": "${DATASOURCE_NAME}",
    "type": "prometheus",
    "isDefault": true,
    "access": "proxy",
    "editable": true,
    "basicAuth": true,
    "basicAuthUser": "${NEW_BASIC_AUTH_USER}",
    "secureJsonData": {
        "basicAuthPassword": "${NEW_BASIC_AUTH_PASSWORD}"
    }
  }
EOF
)

if [ "$dsID" == "null" ]; then
  echo "Datasource '$DATASOURCE_NAME' not found."
  exit 1
fi

http_resp=$(curl -s -o /dev/null -w "%{http_code}" -X PUT -u "${ADMIN_USER}:${ADMIN_PASSWORD}" -H "Accept: application/json" -H "Content-Type: application/json" -d "$UPDATE_DS_PAYLOAD" "http://$HOST:3000/api/datasources/$dsID")

if [ "$http_resp" -eq 200 ]; then
    echo "Datasource '${DATASOURCE_NAME}' update with success."
  else
    echo "Error update datasource '${DATASOURCE_NAME}'. HTTP: $http_resp."
    exit 1
  fi
exit 0

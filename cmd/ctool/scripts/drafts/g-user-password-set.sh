#!/usr/bin/env bash
#
# Copyright (c) 2024 Sigma-Soft, Ltd.
# @author Aleksei Ponomarev
#
# Grafana user password change

set -Eeuo pipefail

set -x

if [[ $# -ne 4 ]]; then
  echo "Usage: $0 <app-node> <grafana admin user> <grafana admin password> <new user password>"
  exit 1
fi


HOST=$1
GRAFANA_HOST="http://$HOST:3000"

# Admin credentials
ADMIN_USER=$2
ADMIN_PASSWORD=$3

# User details
USER_NAME="voedger"
NEW_PASSWORD=$4

# Get user id
user_id=$(curl -s -u "$ADMIN_USER:$ADMIN_PASSWORD" "$GRAFANA_HOST/api/users/lookup?loginOrEmail=$USER_NAME" | jq '.id')

# If not found - error
if [ "$user_id" == "null" ]; then
  echo "User '$USER_NAME' not found."
  exit 1
fi


# Change user password
CHANGE_PASSWORD_PAYLOAD="{\"password\":\"$NEW_PASSWORD\"}"
http_resp=$(curl -s -o /dev/null -w "%{http_code}" -X PUT -H "Content-Type: application/json" -d "$CHANGE_PASSWORD_PAYLOAD" -u "$ADMIN_USER:$ADMIN_PASSWORD" "$GRAFANA_HOST/api/admin/users/$user_id/password")

# Is operation success
if [ "$http_resp" == "200" ]; then
  echo "Password for user '$USER_NAME' changed with success."
else
  echo "Error on password change for user '$USER_NAME'. HTTP response: $http_resp."
  exit 1
fi

exit 0
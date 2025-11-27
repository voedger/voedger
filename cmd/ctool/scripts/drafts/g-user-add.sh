#!/usr/bin/env bash
#
# Copyright (c) 2024 Sigma-Soft, Ltd.
# @author Aleksei Ponomarev
#
# Add new user to grafana

set -Eeuo pipefail

set -x

if [[ $# -ne 3 ]]; then
  echo "Usage: $0 <app-node> <grafana admin user> <grafana admin password>"
  exit 1
fi

HOST=$1
# Admin credentials
ADMIN_USER=$2
ADMIN_PASSWORD=$3

# User details
USER_NAME="voedger"
USER_LOGIN="voedger"
USER_PASSWORD="voedger"

CREATE_USER_PAYLOAD="{\"name\":\"${USER_NAME}\", \
                     \"login\":\"${USER_LOGIN}\", \
                  \"password\":\"${USER_PASSWORD}\"}"

  API_URL="http://$HOST:3000/api/admin/users"

# Check user already exist
user_id=$(curl -s -u "$ADMIN_USER:$ADMIN_PASSWORD" "http://$HOST:3000/api/users/lookup?loginOrEmail=$USER_NAME" | jq '.id')


# If not found - error
if [ "$user_id" == "null" ]; then
    echo "User '$USER_NAME' not found. Create..."
    # Create user using curl and parse response with jq
    resp=$(curl -X POST -H "Content-Type: application/json" -d "$CREATE_USER_PAYLOAD" --user "$ADMIN_USER":"$ADMIN_PASSWORD" "$API_URL")
else
    exit 0
fi

user_id=$(echo "$resp" | jq -r '.id')
message=$(echo "$resp" | jq -r '.message')

if [ "$user_id" != "null" ]; then
  echo "User created with ID: $user_id on $HOST. Message: $message"
else
  echo "User creation failed on $HOST. Message: $message"
  exit 1
fi

exit 0
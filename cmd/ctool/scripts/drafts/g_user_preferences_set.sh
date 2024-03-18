#!/usr/bin/env bash
#
# Copyright (c) 2024 Sigma-Soft, Ltd.
# @author Aleksei Ponomarev
#
# Set preferences for new grafana users

set -euo pipefail

set -x

if [[ $# -ne 3 ]]; then
  echo "Usage: $0 <app-node> <grafana admin user> <grafana admin password>"
  exit 1
fi

HOST=$1
# Admin credentials
ADMIN_USER=$2
ADMIN_PASSWORD=$3

DASHBOARD_ID="HiA8ldL7z"

GLOBAL_PREF_PAYLOAD="{\"theme\":\"dark\", \
           \"homeDashboardUID\":\"$DASHBOARD_ID\", \
                   \"timezone\":\"utc\"}"

# Set preferences on all app nodes hosts
    API_PREFERENCES_URL="http://$HOST:3000/api/org/preferences"
    # need -s
    response=$(curl -X PUT -H "Content-Type: application/json" -d "$GLOBAL_PREF_PAYLOAD" --user "$ADMIN_USER":"$ADMIN_PASSWORD" "$API_PREFERENCES_URL")
    echo "$response"
    # Response must be: {"message":"Preferences updated"}
    msg=$(echo "$response" | jq -r '.message')
    if [ "$msg" != "Preferences updated" ]; then
        echo "Failed to update preferences for the organization"
        exit 1
    fi

exit 0
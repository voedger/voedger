#!/usr/bin/env bash
#
# Copyright (c) 2024 Sigma-Soft, Ltd.
# @author Aleksei Ponomarev
#
# Set preferences for new grafana users

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

DASHBOARD_UID="DMxOsLJSk2"

DashIDbyUID() {
  local dashboard_uid="$1"
  local api_url="http://${HOST}:3000/api/dashboards/uid/${dashboard_uid}"
  local resp
  local dashboard_id

  resp=$(curl -s --user "$ADMIN_USER":"$ADMIN_PASSWORD" "$api_url")
  dashboard_id=$(echo "$resp" | jq '.dashboard.id')

  if [ "$dashboard_id" != "null" ]; then
    echo "$dashboard_id"
  else
    echo "Dashboard с UID $dashboard_uid не найден." >&2
    return 1
  fi
}
DASHBOARD_ID=$(DashIDbyUID $DASHBOARD_UID)

GLOBAL_PREF_PAYLOAD="{\"theme\":\"dark\", \
           \"homeDashboardId\":$DASHBOARD_ID, \
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
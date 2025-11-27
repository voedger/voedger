#!/usr/bin/env bash
#
# Copyright (c) 2024 Sigma-Soft, Ltd.
# @author Aleksei Ponomarev
#
# Update service

set -Eeuo pipefail

set -x

SERVICE_NAME="$1"

if [ -z "$SERVICE_NAME" ]; then
  echo "Usage: $0 <service_name>"
  exit 1
fi

# List all services
services=$(docker service ls --filter "name=$SERVICE_NAME" --format '{{.Name}}')

for service in $services; do
  echo "Checking service: $service"
  tasks=$(docker service ps "$service" --no-trunc --format '{{.ID}} {{.Name}} {{.CurrentState}}')

  while IFS= read -r task; do
    task_id=$(echo "$task" | awk '{print $1}')
    name=$(echo "$task" | awk '{print $2}')
    current_state=$(echo "$task" | awk '{print $3}')

    if [[ $current_state == "Running" ]]; then
      echo "Service $service has a task ($name) that is running ($task_id): restarting..."
      docker service update --force "$service"
    fi
  done <<< "$tasks"
done

set +x

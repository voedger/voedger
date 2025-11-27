#!/usr/bin/env bash
#
# Copyright (c) 2023 Sigma-Soft, Ltd.
# @author Aleksei Ponomarev
#
# Create docker-compose.yml for scylla stack and deploy

set -Eeuo pipefail

set -x

if [[ $# -ne 2 ]]; then
  echo "Usage: $0 <app-node-1> <app-node-2>"
  exit 1
fi

source ./utils.sh

SSH_USER=$LOGNAME

AppNode1=$1
AppNode2=$2

cat ./docker-compose-mon.yml | \
    sed "s/{{.AppNode1}}/app-node-1/g; s/{{.AppNode2}}/app-node-2/g" \
    | utils_ssh "$SSH_USER@$AppNode1" "cat > ~/docker-compose-mon.yml"

utils_ssh "$SSH_USER@$AppNode1" "docker stack deploy --compose-file ~/docker-compose-mon.yml MonDockerStack"


echo "Waiting for services in MonDockerStack to start..."

while true; do
    services=$(utils_ssh "$SSH_USER@$AppNode1" "docker service ls --format '{{.Name}}' | grep 'MonDockerStack'")

    all_running=true
    for service in $services; do
        replicas=$(utils_ssh "$SSH_USER@$AppNode1" "docker service ps --format '{{.CurrentState}}' $service | grep Running | wc -l")

        desired_replicas=$(utils_ssh "$SSH_USER@$AppNode1" "docker service inspect --format '{{.Spec.Mode.Replicated.Replicas}}' $service")

        if [ "$replicas" != "$desired_replicas" ]; then
            all_running=false
            break
        fi
    done

    if [ "$all_running" = true ]; then
        echo "All services in MonDockerStack are running."
        break
    else
        echo "Not all services are running yet. Waiting..."
        sleep 10
    fi
done

set +x

#!/usr/bin/env bash
#
# Copyright (c) 2024 Sigma-Soft, Ltd.
# @author Dmitry Molchanovsky
#
# Restanation of the docker container with name fragment

set -Eeuo pipefail
set -x

if [ $# -ne 1 ]; then
    echo "Usage: $0 <container name>"
    exit 1
fi

# Фрагмент имени контейнера

NAME_FRAGMENT=$1

# Проверяем, что фрагмент имени контейнера передан в качестве аргумента
if [ -z "$NAME_FRAGMENT" ]; then
  echo "Usage: $0 <name_fragment>"
  exit 1
fi

# Получаем список всех контейнеров, содержащих фрагмент имени
CONTAINERS=$(sudo docker ps -a --format '{{.Names}}' | grep "$NAME_FRAGMENT")

# Проверяем, найден ли хотя бы один контейнер
if [ -z "$CONTAINERS" ]; then
  echo "No containers found with name fragment: $NAME_FRAGMENT"
  exit 1
fi

# Рестартуем все найденные контейнеры
for CONTAINER in $CONTAINERS; do
  echo "Restarting container: $CONTAINER"
  sudo docker restart "$CONTAINER"
done

set +x
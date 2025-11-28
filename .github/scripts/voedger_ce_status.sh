#!/usr/bin/env bash
# Copyright (c) 2024 Sigma-Soft, Ltd.
# @author Aleksei Ponomarev

set -Eeuo pipefail

set -x

url=$1
attempts=3
sleep 20

for ((i=1; i<=attempts; i++)); do
    echo "Attempt $i to connect to $url"
    if curl --output /dev/null --fail -Iv "$url"; then
        echo "Website is available over HTTP."
        break
    fi
    if [ "$i" -lt "$attempts" ]; then
        echo "Retrying in 100 seconds..."
        sleep 100
    else
        echo "Maximum attempts reached. Website is not available."
        exit 1
    fi
done

#!/bin/sh

script_dir=$(dirname "$(realpath -s "$0")")

deployments_dir="$script_dir/../deployments"

docker compose -f "$deployments_dir/compose.yml" --profile local-dev up -d --build --force-recreate

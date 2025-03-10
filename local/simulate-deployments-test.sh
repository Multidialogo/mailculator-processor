#!/bin/sh

script_dir=$(dirname $(realpath -s $0))

root_dir="$script_dir/.."

deployments_dir="$root_dir/deployments"

docker compose -f "$deployments_dir/docker/compose.yml" --profile test-deps up -d --build --force-recreate

docker compose -f "$deployments_dir/docker/compose.yml" run --rm test

docker compose -f "$deployments_dir/docker/compose.yml" --profile test-deps down --remove-orphans -v

chown 1000:1000 -R "$root_dir"

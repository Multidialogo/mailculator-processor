#!/bin/sh

script_dir=$(dirname $(realpath -s $0))

root_dir="$script_dir/.."

deployments_dir="$root_dir/deployments"

docker compose -f "$deployments_dir/compose.yml" exec local-dev go mod tidy

docker compose -f "$deployments_dir/compose.yml" exec local-dev sh ./deployments/resources/coverage.sh

sudo chown 1000:1000 -R "$root_dir"

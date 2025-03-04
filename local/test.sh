#!/bin/sh

script_dir=$(dirname $(realpath -s $0))

docker compose -f "$script_dir/docker/compose.yml" exec app go mod tidy
docker compose -f "$script_dir/docker/compose.yml" exec app go test ./...
chown 1000:1000 -R "$script_dir/.."

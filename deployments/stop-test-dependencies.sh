#!/bin/sh

script_dir=$(dirname $(realpath -s $0))

docker_dir="$script_dir/docker"

docker compose -f "$docker_dir/compose.yml" --profile test-deps down --remove-orphans -v
